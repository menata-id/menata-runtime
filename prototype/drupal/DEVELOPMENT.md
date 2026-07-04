# Development Guide — Drupal Prototype

> How to set up Drupal and import the metadata proof config files.

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

### 1. Install Drupal

```bash
composer create-project drupal/recommended-project drupal-menata
cd drupal-menata
```

### 2. Set up PostgreSQL

```sql
CREATE DATABASE menata_drupal;
```

### 3. Install Drupal

```bash
drush site:install standard --db-url=pgsql://postgres:password@localhost/menata_drupal --account-name=admin --account-pass=admin -y
```

### 4. Enable required modules

```bash
drush en workflows eca eca_workflow views -y
```

### 5. Import the metadata proof config files

```bash
cp docs/examples/drupal-config/*.yml web/sites/default/config/sync/
drush config:import -y
```

---

## Running

```bash
drush serve
```

Available at `http://localhost:8888`. Log in with `admin` / `admin`.

---

## Project Structure

```
drupal/
├── docs/
│   ├── drupal-mapping.md
│   ├── decisions/
│   │   └── 001-techstack.md
│   └── examples/
│       ├── design-request.menata           <- Business Knowledge (source)
│       ├── design-request.yaml             <- Runtime Metadata
│       └── drupal-config/                  <- Native Drupal YAML config files
│           ├── node.type.design_request.yml
│           ├── field.storage.node.*.yml
│           ├── workflows.workflow.design_request.yml
│           ├── eca.eca.design_request_notify.yml
│           ├── views.view.design_request_my_requests.yml
│           ├── user.role.requester.yml
│           └── user.role.designer.yml
```

---

## Required Drupal Modules

| Module | Purpose |
|--------|---------|
| workflows | State machine — event-driven status transitions |
| eca | Events, Conditions, Actions — event response execution |
| eca_workflow | ECA integration with Workflow module |
| views | List and detail view realization |

---

## Troubleshooting

**Config import fails**

Run `drush config:import --preview` to see what would change.

**Views not appearing**

Clear cache: `drush cache:rebuild`

**Workflow transitions not working**

Check ECA rules are imported: `drush config:get eca.eca.design_request_notify`
