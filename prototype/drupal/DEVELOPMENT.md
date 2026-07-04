# Development Guide — Drupal Prototype

> This document describes how to set up and run the Menata Runtime Drupal Prototype locally.

---

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| PHP | 8.2+ | Drupal runtime |
| Composer | 2.x | PHP dependency management |
| Drupal | 10.x | Application framework |
| PostgreSQL | 14+ | Database |
| Drush | 12+ | Drupal CLI |

---

## Installation

### 1. Clone the repository

```bash
git clone https://github.com/menata-id/menata.git
cd menata/runtime/prototype/drupal
```

### 2. Install Drupal via Composer

```bash
composer install
```

### 3. Set up PostgreSQL

Create a database:

```sql
CREATE DATABASE menata_drupal;
```

### 4. Configure Drupal

Copy the example settings file:

```bash
cp web/sites/default/example.settings.php web/sites/default/settings.php
```

Edit database connection in `settings.php`:

```php
$databases['default']['default'] = [
  'driver'   => 'pgsql',
  'database' => 'menata_drupal',
  'username' => 'postgres',
  'password' => 'password',
  'host'     => 'localhost',
  'port'     => '5432',
];
```

### 5. Install Drupal

```bash
drush site:install standard --account-name=admin --account-pass=admin -y
```

### 6. Enable required modules

```bash
drush en menata_runtime workflows eca eca_workflow views -y
```

### 7. Realize example Runtime Metadata

```bash
drush menata:realize docs/examples/design-request.yaml
```

---

## Running the Prototype

```bash
drush serve
```

The application will be available at `http://localhost:8888`.

Log in with `admin` / `admin`.

---

## Project Structure

```
drupal/
├── web/
│   ├── modules/
│   │   └── custom/
│   │       └── menata_runtime/       ← custom interpreter module
│   │           ├── menata_runtime.info.yml
│   │           ├── menata_runtime.module
│   │           ├── src/
│   │           │   ├── Interpreter/
│   │           │   │   ├── MachineInterpreter.php    ← Machine → Content Type
│   │           │   │   ├── FieldInterpreter.php      ← Field → Drupal Field
│   │           │   │   ├── EventInterpreter.php      ← Event → Workflow + ECA
│   │           │   │   ├── ConstraintInterpreter.php ← Constraint → Validation
│   │           │   │   ├── PermissionInterpreter.php ← Permission → Roles
│   │           │   │   └── ViewInterpreter.php       ← View → Views module
│   │           │   ├── Validator/
│   │           │   │   └── MetadataValidator.php     ← Runtime Metadata validation
│   │           │   └── Commands/
│   │           │       └── RealizeCommand.php        ← drush menata:realize
│   │           └── tests/
│   └── sites/
│       └── default/
├── composer.json
├── composer.lock
└── docs/
    ├── drupal-mapping.md
    ├── decisions/
    │   └── 001-techstack.md
    └── examples/
        ├── design-request.menata
        └── design-request.yaml
```

---

## The `menata_runtime` Module

The `menata_runtime` module is the interpreter.

It reads Runtime Metadata YAML and realizes it using Drupal's built-in systems.

### Drush Commands

| Command | Description |
|---------|-------------|
| `drush menata:realize <file>` | Realize a Runtime Metadata YAML file |
| `drush menata:validate <file>` | Validate a Runtime Metadata YAML file without applying |
| `drush menata:status` | List realized machines and their status |
| `drush menata:reset <machine_id>` | Remove a realized machine from Drupal |

### Realization Flow

```text
drush menata:realize design-request.yaml
        │
        ▼
MetadataValidator — validate YAML structure
        │
        ▼
MachineInterpreter — create Content Type
        │
        ▼
FieldInterpreter — create Fields
        │
        ▼
EventInterpreter — create Workflow states + ECA rules
        │
        ▼
ConstraintInterpreter — apply Field constraints
        │
        ▼
PermissionInterpreter — assign Permissions to Roles
        │
        ▼
ViewInterpreter — create Views
        │
        ▼
config:export — export as Drupal config YAML
```

---

## Adding a New Machine

1. Define Business Knowledge using Menata Language (see `docs/examples/`)
2. Create Runtime Metadata YAML
3. Run: `drush menata:realize path/to/metadata.yaml`
4. Drupal immediately realizes the new machine as a running application

No custom PHP code is required to add a new machine.

---

## Required Drupal Modules

| Module | Purpose |
|--------|---------|
| `workflows` | State machine for event-driven status transitions |
| `eca` | Events, Conditions, Actions — event response execution |
| `eca_workflow` | ECA integration with Workflow module |
| `views` | List and detail view realization |
| `views_ui` | Views administration (development only) |
| `field_ui` | Field administration (development only) |

---

## Troubleshooting

**`drush menata:realize` fails**

Check validation output. The command validates before applying.

Invalid Runtime Metadata is rejected without modifying Drupal.

**Views not appearing**

Clear Drupal cache: `drush cache:rebuild`

**Workflow transitions not working**

Ensure ECA rules were created correctly: `drush eca:list`
