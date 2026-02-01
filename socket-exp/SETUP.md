# Setup Instructions

## Prerequisites

CGO requires a C compiler. Install one of:

### Option 1: TDM-GCC (Recommended for Windows)
1. Download from: https://jmeubank.github.io/tdm-gcc/
2. Install TDM-GCC 64-bit
3. Add to PATH: `C:\TDM-GCC-64\bin`

### Option 2: MinGW-w64
1. Download from: https://www.mingw-w64.org/downloads/
2. Install and add to PATH

### Verify Installation
```bash
gcc --version
```

## Build

Once GCC is installed:
```bash
build.bat
```

## Alternative: Pre-built Binary

If you can't install GCC, you can:
1. Build on a machine with GCC
2. Copy `socket_lib.dll` and `socket_lib.h` to this directory
3. Use directly with Node.js

## Node.js Setup

```bash
npm install
```

## Test

```bash
node example.js
```
