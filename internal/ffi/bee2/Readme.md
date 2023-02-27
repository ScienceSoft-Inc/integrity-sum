# General overview
Original repo and documentation: https://github.com/agievich/bee2
This repo has included into the main repository as a git submodule.

# Build the bee2 library
Quick build:
- be sure cmake tool is installed
- then run following commands from the root dir of the repository:
    ```bash
    $ mkdir -p bee2/build
    $ cmake -S bee2 -B bee2/build
    $ make -C bee2/build
    ```
- target library (shared and static binary) will be placed at: `./bee2/build/src` dir.

Follow the original documentation for more details.

# Usage
The library may be included into the Go code with an annotation like the following:
```
//#cgo CFLAGS: -I./bee2/include
//#cgo LDFLAGS: -L./bee2/build/src -l:libbee2_static.a
//#include "bee2/crypto/bash.h"
...
```
