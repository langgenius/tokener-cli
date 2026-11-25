# Changelog


## [0.1.7]

### Fixes
* **rx:** embed rx 0.6.2 and drop the permission policy field (#36)
* **agent:** give each agent key a unique name (#38)

### Maintenance
* **repo:** add contribution guardrails (#40)
* **rx:** embed rx 0.1.0 from samzong/rx (#43)
* **cli:** upgrade lathe to v0.7.0-rc.2 (#45)

## [0.1.6]

### Fixes
* **agent:** disable Claude Code experimental betas on launch (#33)
* **rx:** snapshot protocol minor (#34)

### Refactors
* **cli:** consolidate agent and generation plumbing (#32)

## [0.1.5]

### Features
* **agent:** route the gateway endpoint through the selected host (#30)

### Refactors
* **skill:** replace generated skill with tokener-specific instructions

## [0.1.4]

### Documentation
* add Tokener architecture diagram (#25)

### Features
* **models:** include cache price in default list table (#27)

### Fixes
* **keys:** preserve budget precision (#26)
* **agent:** restore interactive harness picker (#28)

## [0.1.3]

### Features
* **cli:** add typed key mutation flags (#18)
* **auth:** add browser device login (#21)
* **cli:** adopt lathe error details, typed create flags, and search terms (#22)

### Fixes
* **cli:** hide key secrets by default and improve CLI ergonomics (#19)

### Maintenance
* **deps:** bump lathe for host error and update fixes (#20)
* **deps:** bump lathe for usage detail coverage (#23)

## [0.1.2]

### Features
* **cli:** improve generated command experience (#8)
* **cli:** improve default table output (#9)
* **cli:** format currency columns (#10)

### Fixes
* **cli:** omit partner commands from tokener (#11)
* **ci:** run rx builds with bash (#13)
* **ci:** restrict rx build permissions (#15)

### Maintenance
* **agent:** automate rx snapshots (#12)
* **agent:** embed rx snapshot f02da0a1ee49 (#16)

## [0.1.1]

### Fixes
* **agent:** resolve launch credentials from bound key only (#5)

### Maintenance
* **agent:** update embedded rx to 0.5.7 (#6)

## [0.1.0]

### Features
* scaffold the tokener CLI generated from the console management spec
* **spec:** source console spec from upstream tag v0.0.2-beta3
* add Tokener agent launcher (#2)
* **release:** add CLI distribution workflow (#3)

### Fixes
* **console:** drop default hostname from generated console specs (#1)

### Maintenance
* **deps:** bump lathe generator from v0.4.5 to v0.6.0
* add cli-install target for local PATH symlink
