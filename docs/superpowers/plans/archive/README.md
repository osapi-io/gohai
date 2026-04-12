# Archived plans

Historical plans superseded by later design decisions. Kept for reference but
should not be executed.

- **2026-04-11-gohai-phase1.md** — original foundation + 3-collector plan.
  Superseded when we moved collectors to `pkg/gohai/collectors/`, adopted the
  library-wrapping methodology (gopsutil, ghw, procfs), and reorganized the
  CLI to follow the osapi `main.go` pattern. The platform collector has since
  been implemented; see `pkg/gohai/collectors/platform/` for the current
  reference implementation.
