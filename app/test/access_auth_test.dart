import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:minder/data/access_auth.dart';

void main() {
  test('PKCE registration, resource and state reach the browser', () async {
    final storage = <String, String>{};
    bool tokenCalled = false;
    final client = MockClient((r) async {
      if (r.url.path == '/.well-known/oauth-authorization-server') {
        return http.Response(
          jsonEncode({
            'authorization_endpoint':
                'https://liveleds.cloudflareaccess.com/auth',
            'token_endpoint': 'https://liveleds.cloudflareaccess.com/token',
            'registration_endpoint':
                'https://liveleds.cloudflareaccess.com/register',
          }),
          200,
        );
      }
      if (r.url.path == '/register') {
        final j = jsonDecode(r.body);
        expect(j['token_endpoint_auth_method'], 'none');
        expect(j['client_uri'], 'https://minder.liveleds.io');
        expect(
          j['logo_uri'],
          'https://minder.liveleds.io/brand/minder-icon.png',
        );
        expect(j['redirect_uris'], ['io.liveleds.minder:/oauth/callback']);
        return http.Response('{"client_id":"native-client"}', 201);
      }
      if (r.url.path == '/token') {
        expect(r.bodyFields['code_verifier']!.length, greaterThanOrEqualTo(43));
        expect(r.bodyFields['resource'], 'https://minder.liveleds.io');
        tokenCalled = true;
        return http.Response(
          '{"access_token":"access","refresh_token":"refresh","expires_in":900}',
          200,
        );
      }
      return http.Response('', 404);
    });
    final auth = AccessAuth(
      'https://minder.liveleds.io',
      client: client,
      read: () => Future.value(storage['auth']),
      write: (v) async {
        storage['auth'] = v;
      },
      browser: (url) async {
        final u = Uri.parse(url);
        expect(u.queryParameters['resource'], 'https://minder.liveleds.io');
        expect(u.queryParameters['code_challenge_method'], 'S256');
        expect(u.queryParameters['code_challenge'], isNotEmpty);
        return 'io.liveleds.minder:/oauth/callback?code=code&state=${u.queryParameters['state']}';
      },
    );
    await auth.signIn();
    expect(tokenCalled, isTrue);
    expect(await auth.headers(), {'Authorization': 'Bearer access'});
    expect(storage['auth'], contains('refresh'));
  });
  test('wrong OAuth state never exchanges a code', () async {
    final client = MockClient((r) async {
      if (r.url.path == '/.well-known/oauth-authorization-server') {
        return http.Response(
          '{"authorization_endpoint":"https://liveleds.cloudflareaccess.com/auth","token_endpoint":"https://liveleds.cloudflareaccess.com/token","registration_endpoint":"https://liveleds.cloudflareaccess.com/register"}',
          200,
        );
      }
      if (r.url.path == '/register') {
        return http.Response('{"client_id":"client"}', 201);
      }
      fail('Invalid callback reached token endpoint');
    });
    final auth = AccessAuth(
      'https://minder.liveleds.io',
      client: client,
      read: () async => null,
      write: (_) async {},
      browser: (_) async =>
          'io.liveleds.minder:/oauth/callback?code=bad&state=wrong',
    );
    await expectLater(auth.signIn(), throwsA(isA<Exception>()));
  });
  test('expired token refreshes without browser sign-in', () async {
    var saved = jsonEncode({
      'client_id': 'c',
      'access_token': 'expired',
      'refresh_token': 'r',
      'expires_at': 0,
      'token_endpoint': 'https://liveleds.cloudflareaccess.com/token',
    });
    final client = MockClient((r) async {
      expect(r.bodyFields['grant_type'], 'refresh_token');
      expect(r.bodyFields['refresh_token'], 'r');
      return http.Response('{"access_token":"fresh","expires_in":900}', 200);
    });
    final auth = AccessAuth(
      'https://minder.liveleds.io',
      client: client,
      read: () async => saved,
      write: (v) async {
        saved = v;
      },
      browser: (_) async {
        fail('Refresh opened browser');
      },
    );
    expect(await auth.headers(), {'Authorization': 'Bearer fresh'});
    expect(saved, contains('"refresh_token":"r"'));
  });
}
