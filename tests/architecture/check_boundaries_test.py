import tempfile
import unittest
import importlib.util
import sys
from pathlib import Path


MODULE_PATH = Path(__file__).with_name("check_boundaries.py")
SPEC = importlib.util.spec_from_file_location("check_boundaries", MODULE_PATH)
check_boundaries = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = check_boundaries
SPEC.loader.exec_module(check_boundaries)


class BoundaryCheckTest(unittest.TestCase):
    def test_parse_imports_supports_single_and_group_imports(self):
        source = '''
package sample

import "fmt"

import (
    alias "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
    _ "github.com/architectcgz/zhicore-go/libs/contracts/events"
)
'''

        imports = check_boundaries.parse_go_imports(source)

        self.assertEqual(imports, [
            "fmt",
            "github.com/architectcgz/zhicore-go/libs/kit/httpapi",
            "github.com/architectcgz/zhicore-go/libs/contracts/events",
        ])

    def test_detects_cross_service_internal_imports(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_go_file(
                root / "services/zhicore-user/internal/user/usecase.go",
                '''
package user

import "github.com/architectcgz/zhicore-go/services/zhicore-auth/internal/auth"
''',
            )

            violations = check_boundaries.check_root(root)

        self.assertViolation(
            violations,
            "cross-service-internal-import",
            "services/zhicore-user/internal/user/usecase.go",
        )

    def test_detects_forbidden_layer_imports(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_go_file(
                root / "services/zhicore-content/internal/content/domain/post.go",
                '''
package post

import "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
''',
            )
            write_go_file(
                root / "services/zhicore-content/internal/content/domain/comment/comment.go",
                '''
package comment

import "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain/post"
''',
            )

            violations = check_boundaries.check_root(root)

        self.assertViolation(
            violations,
            "service-layer-import-not-allowed",
            "services/zhicore-content/internal/content/domain/post.go",
        )
        self.assertViolation(
            violations,
            "service-layer-import-not-allowed",
            "services/zhicore-content/internal/content/domain/comment/comment.go",
        )

    def test_allows_documented_forward_dependencies(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_go_file(
                root / "services/zhicore-content/api/http/handler.go",
                '''
package httpapi

import "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
''',
            )
            write_go_file(
                root / "services/zhicore-content/internal/content/application/service.go",
                '''
package application

import (
    "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
    "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)
''',
            )
            write_go_file(
                root / "services/zhicore-content/internal/content/infrastructure/postgres/repository.go",
                '''
package postgres

import (
    "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
    "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)
''',
            )
            write_go_file(
                root / "services/zhicore-content/internal/content/infrastructure/rabbitmq/consumer.go",
                '''
package rabbitmq

import "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
''',
            )
            write_go_file(
                root / "services/zhicore-content/internal/content/runtime/module.go",
                '''
package runtime

import (
    "github.com/architectcgz/zhicore-go/services/zhicore-content/api/http"
    "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
    "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/infrastructure/postgres"
)
''',
            )

            violations = check_boundaries.check_root(root)

        self.assertEqual([], violations)

    def test_detects_service_layer_imports_outside_allowed_matrix(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_go_file(
                root / "services/zhicore-content/api/http/handler.go",
                '''
package httpapi

import "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
''',
            )
            write_go_file(
                root / "services/zhicore-content/internal/content/application/service.go",
                '''
package application

import "github.com/architectcgz/zhicore-go/services/zhicore-content/api/http"
''',
            )
            write_go_file(
                root / "services/zhicore-content/internal/content/ports/repository.go",
                '''
package ports

import "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/infrastructure/postgres"
''',
            )

            violations = check_boundaries.check_root(root)

        self.assertViolation(
            violations,
            "service-layer-import-not-allowed",
            "services/zhicore-content/api/http/handler.go",
        )
        self.assertViolation(
            violations,
            "service-layer-import-not-allowed",
            "services/zhicore-content/internal/content/application/service.go",
        )
        self.assertViolation(
            violations,
            "service-layer-import-not-allowed",
            "services/zhicore-content/internal/content/ports/repository.go",
        )

    def test_ignores_go_test_files(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_go_file(
                root / "services/zhicore-content/api/http/handler_test.go",
                '''
package httpapi_test

import "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/infrastructure/postgres"
''',
            )

            violations = check_boundaries.check_root(root)

        self.assertEqual([], violations)

    def test_detects_shared_library_importing_service_private_code(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_go_file(
                root / "libs/kit/httpapi/response.go",
                '''
package httpapi

import "github.com/architectcgz/zhicore-go/services/zhicore-upload/internal/upload/application"
''',
            )
            write_go_file(
                root / "libs/contracts/events/content/post.go",
                '''
package content

import "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
''',
            )

            violations = check_boundaries.check_root(root)

        self.assertViolation(
            violations,
            "shared-library-imports-service",
            "libs/kit/httpapi/response.go",
        )
        self.assertViolation(
            violations,
            "shared-library-imports-service",
            "libs/contracts/events/content/post.go",
        )

    def assertViolation(self, violations, rule, path):
        if not any(v.rule == rule and v.path.as_posix().endswith(path) for v in violations):
            formatted = "\n".join(str(v) for v in violations)
            self.fail(f"missing violation {rule} for {path}; got:\n{formatted}")


def write_go_file(path, content):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content.strip() + "\n", encoding="utf-8")


if __name__ == "__main__":
    unittest.main()
