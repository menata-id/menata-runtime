-- 019_process_overlay.sql
-- Process Overlay B1 (brd-menata-runtime-v2.md §7.2, Study 20 §6): a Machine
-- may declare a `process` block -- states, transitions (with actor), auto
-- transitions -- that the metadata loader COMPILES into ordinary Events,
-- guards, and Permissions at load time. The runtime itself never sees the
-- process: it executes the compiled primitives exactly as if they had been
-- hand-authored ("declared process, emergent execution").
--
-- One JSONB column on machines, same shape decision as machines.config
-- (migrations/004): the declaration is a document about the Machine itself,
-- not a Field of its records, and needs no relational structure of its own
-- -- it is consumed whole by the compiler at boot.

ALTER TABLE machines ADD COLUMN IF NOT EXISTS process JSONB;

COMMENT ON COLUMN machines.process IS
  'Process Overlay declaration (B1: states/transitions/actor/auto) -- compiled by the loader into Events/guards/Permissions; NULL = no overlay, the default';
