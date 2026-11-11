import 'dart:convert';
import 'dart:math';

import 'package:crypto/crypto.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_web_auth_2/flutter_web_auth_2.dart';
import 'package:http/http.dart' as http;

import 'web_redirect_stub.dart'
    if (dart.library.js_interop) 'web_redirect_web.dart';
import '../domain/chat_repository.dart';

class AccessAuth {
  AccessAuth(
    this.base, {
    http.Client? client,
    Future<String?> Function()? read,
    Future<void> Function(String)? write,
    Future<String> Function(String)? browser,
  }) : _client = client ?? http.Client(),
       _read =
           read ??
           (() => const FlutterSecureStorage().read(key: 'minder.access')),
       _write =
           write ??
           ((v) => const FlutterSecureStorage().write(
             key: 'minder.access',
             value: v,
           )),
       _browser =
           browser ??
           ((url) => FlutterWebAuth2.authenticate(
             url: url,
             callbackUrlScheme: 'io.liveleds.minder',
           ));
  final String base;
  final http.Client _client;
  final Future<String?> Function() _read;
  final Future<void> Function(String) _write;
  final Future<String> Function(String) _browser;
  static const issuer = 'https://liveleds.cloudflareaccess.com';
  static const redirect = 'io.liveleds.minder:/oauth/callback';
  Future<Map<String, dynamic>>? _refreshing;
  String _random() {
    final r = Random.secure();
    return base64Url
        .encode(List.generate(32, (_) => r.nextInt(256)))
        .replaceAll('=', '');
  }

  Future<Map<String, dynamic>> _json(http.Response r) {
    if (r.statusCode < 200 || r.statusCode >= 300) {
      throw const ApiException('Sign in to continue.', unauthorized: true);
    }
    return Future.value(jsonDecode(r.body) as Map<String, dynamic>);
  }

  Uri _endpoint(String value) {
    final u = Uri.parse(value);
    if (u.scheme != 'https' || u.host != 'liveleds.cloudflareaccess.com') {
      throw const ApiException('Sign-in server is not trusted.');
    }
    return u;
  }

  Future<void> signIn() async {
    final meta = await _json(
      await _client
          .get(Uri.parse('$issuer/.well-known/oauth-authorization-server'))
          .timeout(const Duration(seconds: 20)),
    );
    final registration = await _json(
      await _client
          .post(
            _endpoint(meta['registration_endpoint'] as String),
            headers: {'Content-Type': 'application/json'},
            body: jsonEncode({
              'client_name': 'Minder Android',
              'client_uri': base,
              'logo_uri': '$base/brand/minder-icon.png',
              'redirect_uris': [redirect],
              'grant_types': ['authorization_code', 'refresh_token'],
              'response_types': ['code'],
              'token_endpoint_auth_method': 'none',
            }),
          )
          .timeout(const Duration(seconds: 20)),
    );
    final verifier = _random(), state = _random();
    final challenge = base64Url
        .encode(sha256.convert(ascii.encode(verifier)).bytes)
        .replaceAll('=', '');
    final url = _endpoint(meta['authorization_endpoint'] as String).replace(
      queryParameters: {
        'response_type': 'code',
        'client_id': registration['client_id'] as String,
        'redirect_uri': redirect,
        'code_challenge': challenge,
        'code_challenge_method': 'S256',
        'state': state,
        'resource': base,
        'scope': 'openid email offline_access',
      },
    );
    final result = Uri.parse(await _browser(url.toString()));
    if (result.scheme != 'io.liveleds.minder' ||
        result.path != '/oauth/callback' ||
        result.queryParameters['state'] != state ||
        result.queryParameters['code'] == null) {
      throw const ApiException('Sign-in did not finish. Try again.');
    }
    final endpoint = _endpoint(meta['token_endpoint'] as String);
    final token = await _json(
      await _client
          .post(
            endpoint,
            body: {
              'grant_type': 'authorization_code',
              'client_id': registration['client_id'] as String,
              'code': result.queryParameters['code']!,
              'code_verifier': verifier,
              'redirect_uri': redirect,
              'resource': base,
            },
          )
          .timeout(const Duration(seconds: 20)),
    );
    await _save({
      ...token,
      'client_id': registration['client_id'],
      'token_endpoint': endpoint.toString(),
    });
  }

  Future<void> _save(Map<String, dynamic> data) async {
    if (data['access_token'] is! String) {
      throw const ApiException('Sign-in returned no token.');
    }
    data['expires_at'] =
        DateTime.now().millisecondsSinceEpoch +
        ((data['expires_in'] as num?)?.toInt() ?? 900) * 1000;
    await _write(jsonEncode(data));
  }

  Future<Map<String, dynamic>> _refresh(Map<String, dynamic> data) async {
    final endpoint = _endpoint(data['token_endpoint'] as String);
    if (data['refresh_token'] == null) {
      throw const ApiException('Sign in to continue.', unauthorized: true);
    }
    final fresh = await _json(
      await _client
          .post(
            endpoint,
            body: {
              'grant_type': 'refresh_token',
              'client_id': data['client_id'] as String,
              'refresh_token': data['refresh_token'] as String,
              'resource': base,
            },
          )
          .timeout(const Duration(seconds: 20)),
    );
    final merged = {...data, ...fresh};
    await _save(merged);
    return merged;
  }

  Future<Map<String, String>> headers() async {
    final raw = await _read();
    if (raw == null) {
      throw const ApiException('Sign in to continue.', unauthorized: true);
    }
    var data = jsonDecode(raw) as Map<String, dynamic>;
    if ((data['expires_at'] as num).toInt() <
        DateTime.now().millisecondsSinceEpoch + 60000) {
      try {
        _refreshing ??= _refresh(data);
        data = await _refreshing!;
      } finally {
        _refreshing = null;
      }
    }
    return {'Authorization': 'Bearer ${data['access_token']}'};
  }

  Future<void> webSignIn() async => redirectWeb(base);
}
