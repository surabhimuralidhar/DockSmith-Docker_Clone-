# DockSmith-Docker_Clone-

A Simplified Container Engine from Scratch

Overview

Docksmith is a minimal, functional clone of Docker built to explore the
low-level internals of containerization. It focuses on three technical pillars:
immutable filesystem layering, deterministic build caching, and native
Linux process isolation.

Key Achievement: This project implements process sandboxing using
Linux Namespaces and Chroot directly via Go's syscall package,
without relying on external tools like unshare or Docker.

🛠 Architecture & Storage

All state is stored locally in the ~/.docksmith/ directory on the host
machine.
/images : JSON manifests defining the order of layers and runtime
config.
/layers : Content-addressed .tar files named by their SHA256
digest.
/cache : An index mapping build instructions to pre-computed layer
digests.
•

•

•

🏗 Build Instructions

Instruction Behavior

FROM Loads a base image from the local store.
COPY Adds host files to a new deterministic layer.

RUN

Executes commands in a sandbox and captures
filesystem deltas.

WORKDIR Sets the active directory for subsequent steps.
ENV Defines persistent environment variables.
CMD Specifies the default execution command.

Usage Guide

Build an Image

sudo ./docksmith build -t myapp:latest ./examples/simple-app

Run with Environment Override

sudo ./docksmith run -e MESSAGE="Hello World" myapp:latest

Strict Isolation Test
Verify that the container is truly sandboxed by creating a file inside it and
checking the host:

sudo ./docksmith run myapp:latest -- sh -c "touch /tmp/isolated.txt"
ls /tmp/isolated.txt # Should return 'No such file'

Compliance

This engine fulfills the "Hard Pass/Fail" requirement for process isolation.
By using CLONE_NEWPID and CLONE_NEWNS , processes are successfully
trapped within their unique root filesystem.
