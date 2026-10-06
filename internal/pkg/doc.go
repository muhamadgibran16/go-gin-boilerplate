// Package pkg is the home of generic utilities shared by many modules.
//
// Rules:
//   - One sub-package per concern, named after what it provides
//     (pagination, password, stringx, ...). Never a catch-all "utils" package.
//   - Code here must not know about any module: it must not import internal/modules
//     or internal/app. Modules import pkg, never the other way around.
//   - Only move code here once a second module needs it; until then keep it in the module.
package pkg
