import pathlib
import runpy
import tempfile
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[2]


class ArchitectureChecks(unittest.TestCase):
    def test_application_cannot_import_storage_or_http(self):
        checks = runpy.run_path(str(ROOT / 'toolchain/architecture-check'))
        errors = checks['go_errors']([
            {'ImportPath': 'github.com/alexanderp4580/Minder/server/command',
             'Imports': ['github.com/alexanderp4580/Minder/server/store', 'net/http/httptest']},
        ])
        self.assertEqual(len(errors), 2)

    def test_application_can_import_domain_and_ports(self):
        checks = runpy.run_path(str(ROOT / 'toolchain/architecture-check'))
        errors = checks['go_errors']([
            {'ImportPath': 'github.com/alexanderp4580/Minder/server/command',
             'Imports': ['github.com/alexanderp4580/Minder/server/domain',
                         'github.com/alexanderp4580/Minder/server/port', 'time']},
        ])
        self.assertEqual(errors, [])

    def test_flutter_domain_and_controllers_reject_transport_imports(self):
        checks = runpy.run_path(str(ROOT / 'toolchain/architecture-check'))
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            (root / 'app/lib/domain').mkdir(parents=True)
            (root / 'app/lib/application').mkdir()
            (root / 'app/lib/domain/item.dart').write_text("import 'dart:io';\n")
            (root / 'app/lib/application/chat.dart').write_text("import '../data/api.dart';\n")
            self.assertEqual(len(checks['dart_errors'](root)), 2)


    def test_flutter_conditional_import_cannot_bypass_the_layer_boundary(self):
        checks = runpy.run_path(str(ROOT / 'toolchain/architecture-check'))
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            (root / 'app/lib/domain').mkdir(parents=True)
            (root / 'app/lib/domain/item.dart').write_text(
                "import 'item_stub.dart' if (dart.library.io) '../data/api.dart';\n")
            self.assertEqual(len(checks['dart_errors'](root)), 1)



class DoxChecks(unittest.TestCase):
    def make_tree(self, root):
        (root / 'AGENTS.md').write_text('# Child DOX Index\n\n- `server/AGENTS.md`: server\n')
        server = root / 'server'
        server.mkdir()
        (server / 'AGENTS.md').write_text(self.doc('None.'))
        return server

    @staticmethod
    def doc(index):
        sections = ['Purpose', 'Ownership', 'Local Contracts', 'Work Guidance', 'Verification', 'Child DOX Index']
        return '\n\n'.join('# ' + name + '\n\n' + (index if name == 'Child DOX Index' else 'Current contract.') for name in sections)

    def test_complete_tree_passes(self):
        check = runpy.run_path(str(ROOT / 'toolchain/dox-check'))['check']
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            self.make_tree(root)
            self.assertEqual(check(root), [])

    def test_new_child_requires_its_parent_index(self):
        check = runpy.run_path(str(ROOT / 'toolchain/dox-check'))['check']
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            server = self.make_tree(root)
            (server / 'command').mkdir()
            (server / 'command/AGENTS.md').write_text(self.doc('None.'))
            self.assertTrue(any('index' in error for error in check(root)))

    def test_go_package_requires_a_local_owner(self):
        check = runpy.run_path(str(ROOT / 'toolchain/dox-check'))['check']
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            server = self.make_tree(root)
            (server / 'command').mkdir()
            (server / 'command/service.go').write_text('package command\n')
            self.assertTrue(any('owner' in error for error in check(root)))


if __name__ == '__main__':
    unittest.main()
