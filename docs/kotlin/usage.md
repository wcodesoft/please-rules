# Kotlin Usage Guide

This guide walks through configuring and using the Please Kotlin plugin in your
project.

---

## 1. Plugin Declaration

Declare the Kotlin plugin in your project's `plugins/BUILD`:

```starlark
plugin_repo(
    name = "kotlin",
    owner = "wcodesoft",
    plugin = "please-rules",
    revision = "kotlin-v0.1.0",     # or revision = "kotlin"
)
```

---

## 2. Toolchain Configuration

### Setting up `third_party/kotlin/BUILD`

```starlark
subinclude("///kotlin//build_defs:kotlin")

kotlin_toolchain(
    name = "toolchain",
    version = "2.1.10",
    visibility = ["PUBLIC"],
)
```

### Configuring `.plzconfig`

Bind the plugin to the hermetic toolchain in `.plzconfig`:

```ini
[Plugin "kotlin"]
Target = //plugins:kotlin
KotlincTool = //third_party/kotlin:toolchain|kotlinc
JavaTool = //third_party/kotlin:toolchain|java
JacocoAgent = //third_party/kotlin:toolchain|jacoco-agent
JacocoCli = //third_party/kotlin:toolchain|jacoco-cli
JvmTarget = 21

[cover]
fileextension = .kt
```

---

## 3. Building and Testing

### Build targets

```bash
./pleasew build //src/...
```

### Run an application

```bash
./pleasew run //src/app:bin
```

### Run tests with JUnit results

```bash
./pleasew test //src/...
```

### Run tests with line-by-line coverage

```bash
./pleasew cover //src/...
```
