#!/usr/bin/env python3
"""Export canonical orbital SVG artwork to Flutter and inline guide previews."""
import argparse
import copy
import math
from pathlib import Path
import re
import xml.etree.ElementTree as ET

ROOT = Path(__file__).resolve().parent.parent
NS = {'s': 'http://www.w3.org/2000/svg'}
ET.register_namespace('', NS['s'])
NUMBER = r'-?(?:\d+(?:\.\d*)?|\.\d+)'


def number(value):
    return repr(float(value))


def read_scene(scene):
    svg = ET.parse(ROOT / f'brand/orbital-{scene}.svg').getroot()
    assert svg.attrib['viewBox'] == '0 0 462 868', 'Keep the reference coordinate system fixed'
    paths = svg.find('s:g[@id="tracks"]', NS)
    lights = svg.find('s:g[@id="atmosphere-lights"]', NS)
    assert paths is not None and lights is not None, 'Missing artwork layers'
    background = svg.find('s:rect', NS)
    assert background is not None and background.attrib.get('fill') == '#0a0a0a', 'Artwork background uses Void colour'
    # Keep the shared renderer profile explicit; reject source changes we cannot export.
    assert paths.attrib.get('stroke-width') == '.85', 'Renderer edge width is .85'
    assert all(stop.attrib.get('stop-color') == '#f2f2f2' for stop in svg.findall('.//s:stop', NS)), 'Orbital artwork uses Text colour'
    atmosphere = svg.find('.//s:radialGradient[@id="atmosphere"]', NS)
    assert atmosphere is not None and set(atmosphere.attrib) == {'id'}, 'Use centred atmosphere gradients'
    assert [(float(stop.attrib.get('offset', 0)), float(stop.attrib['stop-opacity'])) for stop in atmosphere] == [(0, 1), (.4, .22), (1, 0)], 'Renderer atmosphere profile differs'
    glow = svg.find('s:g[@id="edge-glow"]', NS)
    assert glow is not None and float(glow.attrib['opacity']) == .16 and float(glow.attrib['stroke-width']) == 2, 'Renderer glow profile differs'
    blur = svg.find('.//s:feGaussianBlur', NS)
    assert blur is not None and float(blur.attrib['stdDeviation']) == 3.5, 'Renderer glow blur is 3.5'
    assert [item.attrib['href'] for item in glow] == ['#' + path.attrib['id'] for path in paths], 'Glow must reuse the source tracks'
    commands = []
    edges = []
    for path in paths:
        assert set(path.attrib) == {'id', 'd', 'fill', 'stroke'} and path.attrib['fill'] == 'none', 'Keep source tracks unfilled with inherited width'
        tokens = re.findall(r'[MCZ]|' + NUMBER, path.attrib['d'])
        assert not re.sub(r'[MCZ\s,]|' + NUMBER, '', path.attrib['d']), 'Use absolute M/C/Z paths'
        calls = ['Path()']
        while tokens:
            command = tokens.pop(0)
            method, count = {'M': ('moveTo', 2), 'C': ('cubicTo', 6), 'Z': ('close', 0)}[command]
            values, tokens = tokens[:count], tokens[count:]
            calls.append('..' + method + '(' + ', '.join(map(number, values)) + ')')
        commands.append(''.join(calls))
        gradient_id = path.attrib['stroke'][5:-1]
        gradient = svg.find(f'.//s:linearGradient[@id="{gradient_id}"]', NS)
        assert gradient is not None
        start = [number(gradient.attrib[key]) for key in ['x1', 'y1']]
        end = [number(gradient.attrib[key]) for key in ['x2', 'y2']]
        stops = ', '.join('(' + number(stop.attrib['offset']) + ', ' + number(stop.attrib['stop-opacity']) + ')' for stop in gradient)
        edges.append('(start: const Offset(' + ', '.join(start) + '), end: const Offset(' + ', '.join(end) + '), stops: const [' + stops + '])')
    light_data = []
    for light in lights:
        rotation = re.fullmatch(r'rotate\((' + NUMBER + r') (' + NUMBER + r') (' + NUMBER + r')\)', light.attrib['transform'])
        assert rotation is not None and float(rotation[2]) == float(light.attrib['cx']) and float(rotation[3]) == float(light.attrib['cy']), 'Rotate atmosphere lights about their centres'
        assert light.attrib['fill'] == 'url(#atmosphere)', 'Use the shared atmosphere profile'
        tilt = float(rotation[1]) * math.pi / 180
        values = {key: light.attrib[attr] for key, attr in [('x', 'cx'), ('y', 'cy'), ('rx', 'rx'), ('ry', 'ry'), ('alpha', 'opacity')]}
        values['tilt'] = tilt
        light_data.append('(' + ', '.join(key + ': ' + number(value) for key, value in values.items()) + ')')
    dart = f'OrbitalData {scene}Artwork() => (\n  paths: [\n    ' + ',\n    '.join(commands) + ',\n  ],\n  edges: [\n    ' + ',\n    '.join(edges) + ',\n  ],\n  lights: const [\n    ' + ',\n    '.join(light_data) + ',\n  ],\n);\n'
    # The guide renders the same SVG, with travelling highlights as a separate layer.
    svg.set('class', f'orbit-artwork orbit-{scene}')
    svg.set('preserveAspectRatio', 'xMidYMin slice')
    svg.set('aria-hidden', 'true')
    travel = ET.SubElement(svg, '{' + NS['s'] + '}g', {'class': 'orbit-travel'})
    for index, path in enumerate(paths):
        item = copy.deepcopy(path)
        item.attrib.update({'id': f'travel-{scene}-{index}', 'stroke': '#f2f2f2', 'stroke-width': '1.1', 'opacity': '.32', 'pathLength': '1000', 'stroke-dasharray': '35 965', 'style': f'animation-delay:-{index * 36}s'})
        travel.append(item)
    return dart, ET.tostring(svg, encoding='unicode')


def update(path, expected, check):
    target = ROOT / path
    if target.exists() and target.read_text() == expected:
        return True
    if check:
        print(f'STALE orbital export: {path}')
        return False
    target.write_text(expected)
    return True


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--check', action='store_true')
    args = parser.parse_args()
    output = '''// Generated by toolchain/export-orbits.py from brand/orbital-*.svg. Do not edit.
// dart format off
import 'dart:ui';

typedef OrbitalLight = ({double x, double y, double rx, double ry, double alpha, double tilt});
typedef OrbitalEdge = ({Offset start, Offset end, List<(double, double)> stops});
typedef OrbitalData = ({List<Path> paths, List<OrbitalEdge> edges, List<OrbitalLight> lights});

'''
    markup = {}
    for scene in ['calendar', 'chat']:
        dart, svg = read_scene(scene)
        output += dart + '\n'
        markup[scene] = svg
    output += '// dart format on\n'
    valid = update('app/lib/ui/orbital_artwork_data.dart', output, args.check)
    for path, scene in [('brand/index.html', 'calendar'), ('docs/mockups/calendar.html', 'calendar'), ('docs/mockups/chat.html', 'chat')]:
        text = (ROOT / path).read_text()
        pattern = r'<svg[^>]*class="orbit-(?:lines|artwork) orbit-' + scene + r'".*?</svg>'
        expected, count = re.subn(pattern, lambda _: markup[scene], text, flags=re.S)
        assert count == 1, f'Expected one orbital preview in {path}'
        valid = update(path, expected, args.check) and valid
    if not valid:
        return 1
    print('PASS: orbital SVG, Flutter artwork and guide/mockup exports agree.')
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
