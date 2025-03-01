# Asterisc

Asterisc proves execution of a RISC-V program with an interactive fraud-proof.

*Deploy/run this at your own risk, this is highly experimental software*

## Work in progress

This project is a work in progress. Maybe 80% complete.

TODO
- [ ] Go:
  - [x] Implement yul opcodes in Go
    - [x] fast/slow u256
    - [x] fast u64
    - [x] slow u64 (u256 backed)
  - [x] Implement merkleized state loading/storing state-machine
  - [x] Implement ALU + load/store parts of RISC-V
  - [x] Implement state merkleization
  - [x] Implement access-list collecter shim and db
  - [ ] update state model to track multiple copies of registers state and PC
  - [x] change Go step code to use uint256 words
  - [x] split Go into fast and slow mode
  - [ ] support syscalls
    - [x] memory brk/mmap
    - [x] exit
    - [x] read/write/fcntl/openat
    - [x] Go syscalls compat
    - [ ] extend w/ threading clone/futex/gettid/tgkil/tkill
    - [ ] extras
  - [x] read/write syscall based pre-image oracle
  - [x] Pass RISC-V test vectors (RV64 I, M, A)
- [ ] Sol:
  - [x] Forge solidity testing setup
  - [ ] Forge solidity test VM step on snapshotted proof data
  - [x] Complete port of Go slow-mode emu to solidity/Yul
  - [x] Pass RISC-V test vectors
  - [x] read/write syscall based pre-image oracle
- [ ] Misc:
  - [x] analyze Go runtime/compiler
  - [ ] CLI command to build VM snapshot from ELF
  - [ ] CLI command to transition VM snapshot N steps with interval snapshots
  - [ ] CLI command to produce proof for snapshot
  - [ ] CLI command to verify proof, in EVM and Go mode
  - [ ] Go-Sol differential fuzzing
  - [x] Test basic Go programs

## How does it work?

Interactively different parties can agree on a common prefix of the execution trace states,
and the execution first step where things are different.

Asterisc produces the commitments to the memory, register, CSR and VM state during the execution trace,
and emulates the disputed step inside the EVM to determine the correct state after.

Asterisc consists of two parts:
- `rvgo`: Go implementation of Risc-V emulator with tooling
  - Fast-mode: Emulate 1 instruction per step, directly on a VM state
  - Slow-mode: Emulate 1 instruction per step, on a VM state-oracle.
  - Tooling: merkleize VM state, collect access-list of a slow-mode step, diff VM merkle-trees
- `rvsol`: Solidity/Yul mirror of Go Risc-V slow-mode step that runs with access-list as input

All VM state is merkleized into a single big structured binary merkle-trie.

### Why use Yul in solidity?

The use of YUL / "solidity assembly" is very convenient because:
- it offers pretty `switch` statements
- it uses function calls, no operators, that can be mirrored exactly in Go
- it preserves underflow/overflow behavior: this is a feature, not a bug, when emulating an ALU and registers.
- it operates on unsigned word-sized integers only: no typed data, just like registers don't have types.
  - In Go it is typed `U64` and `U256` for sanity, but it's all `uint256` words in slow mode.

### Why fast and slow mode?

Emulating a program on top of merkleized key-value backed memory is expensive.
When bisecting a program trace, you only need to produce a commitment to a few intermediate states, not all of them.

Note that slow mode and fast mode ALUs can be implemented exactly the same, just with different u64/u256 implementations.
The slow mode matches the smart-contract behavior 1:1 and is useful for building the access-list
and having a Go mirror of the smart-contract behavior for testing/debugging in general.

## RISC-V subset support

- `RV32I` support - 32 bit base instruction set
  - `FENCE`, `ECALL`, `EBREAK` are hardwired to implement a minimal subset of systemcalls of the linux kernel
    - Work in progress. All syscalls used by the Golang `risc64` runtime. 
- `RV64I` support
- `RV32M`+`RV64M`: Multiplication support
- `RV32A`+`RV64A`: Atomics support
- `RV{32,64}{D,F,Q}`: no-op: No floating points support (since no IEEE754 determinism with rounding modes etc., nor worth the complexity)
- `Zifencei`: `FENCE.I` no-op: No need for `FENCE.I`
- `Zicsr`: no-op: some support for Control-and-status registers may come later though.
- `Ztso`: no-op: no need for Total Store Ordering
- other: revert with error code on unrecognized instructions

Where necessary, the non-supported operations are no-ops that allow execution of the standard Go runtime, with disabled GC.

## Contributing

The primary purpose of Asterisc is to run a Go program to fraud-proof an optimistic rollup.

This program can include the go-ethereum EVM for an EVM-equivalent rollup fraud proof,
but may also be a totally different RISC-V program.

If you are one of these bespoke other programs to fraud-proof, please upstream fixes,
but do not expect support if you diverge from the general design direction:
- Simplicity and security with minimalism
- First-class Golang support
- Mirror Solidity and Go step implementations

Asterisc may be usable to fraud-proof Rust programs or bespoke execution-environments in the future,
but doing so should stay stupid-simple & not negatively affect its primary purpose.

## Asterisc status

This project is not (yet) a production-ready fault-proof system, and is developed during spare-time only.

The end-game (pre-ZK) is for Ethereum L2 optimistic rollups to embed multiple fraud-proof modules to function as a "committee":
if one of the members is corrupted due to a bug/vulnerability, then the system as a whole stays stable without rollbacks or human intervention.
So Asterisc aims to complement other fraud-proof systems, and not to replace them.

## Docs

- [Go support](./docs/golang.md): relevant info about the Go runtime / compiler to support it
- [RISC-V resources](./docs/riscv.md): RISC-V instruction set specs, references and notes
- [Toolchain notes](./docs/toolchain.md): RISC-V and Go toolchain help

## Why not X?

### Why not Cannon?

[Cannon](https://github.com/ethereum-optimism/cannon/), originally by [`geohot`](https://github.com/geohot/) and
now maintained by Optimism does the same thing, but differently.
Cannon is 32-bit, runs MIPS, does not support threads, and memory reads/writes happen in a single execution step.

Asterisc aims to be an alternative to this, and more future-compatible: RISC-V is gaining adoption unlike MIPS,
and with 64 bit operations and concurrent but deterministic threads it may support more programs.

### Why not Cartesi?

[Cartesi](https://github.com/cartesi/) has a much larger scope of RISC-V fraud-proving a full machine,
including a lot more features. More features = more complexity however, which can form a risk.

Asterisc aims to be more minimal, simple and easy to audit. By running a single process,
and hardwiring only the necessary systemcalls, the complexity of supporting all the RISC-V instruction set extensions 
and running a full linux kernel or multi-process system is avoided.

### Why not web-assembly?

A fraud-proof of Go with a web-assembly runtime is already being developed by Arbitrum,
although with a business-source license and with a transformation to "WAVM":
not generic enough to use it for other purposes.

Asterisc aims to be open for anyone to use with MIT license.

## License

MIT, see [`LICENSE` file](./LICENSE)
