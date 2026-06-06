---
name: runtime-ops-safety
description: Use when running tests, scripts, recursive scans, background jobs, or performance checks that may consume significant resources, outlive the current turn, or leave residual processes.
---

# Runtime Ops Safety

## Overview

Protect machine stability, result validity, and session hygiene when executing commands.

## When to Use

- Running tests, scripts, or tools that may continue after the current reply
- Starting background processes, temporary servers, tunnels, or long-lived shells
- Running recursive `find`, `du`, or large scans, especially on `/mnt/c`, `/mnt/d`, or other mounted Windows drives
- Executing performance, load, soak, or resource-sensitive validation
- Performing multi-step verification where later steps depend on earlier state

## Core Rules

### 1. Bound the command before running it

- Prefer the smallest scope that can answer the question.
- Prefer metadata, targeted subpaths, or service-native commands over blind recursive scans.
- Add explicit timeouts whenever practical.
- Treat `/mnt/c`, `/mnt/d`, and similar mounted drives as slow and interruption-prone. Avoid broad `find` and `du` unless no smaller path can answer the question.

### 2. Keep dependent verification steps serial

- If step B depends on step A's side effect, do not parallelize them.
- Typical serial chains include:
  - `delete key -> warm up -> inspect TTL`
  - `build -> deploy -> health check`
  - `prepare -> pressure test -> sample -> retest`

### 3. Keep performance conclusions defensible

- Compare only samples that differ by one variable at a time.
- Start with short, low-risk smoke runs before increasing duration or concurrency.
- Before raising pressure, check host memory, swap, load average, and whether previous test processes were cleaned up.
- Stop immediately if the host becomes unstable, swap keeps growing, the tool becomes the bottleneck, or output is incomplete.

### 4. Clean up what you start

- After tests, scans, or temporary orchestration, shut down background processes, temporary servers, tunnels, and extra shells unless the user explicitly asks to keep them.
- If the user interrupts the turn or reports `background terminals running`, immediately inspect for residual child processes and terminate the ones created for the task.
- Do not leave idle terminals or orphaned scans behind.

## Quick Checklist

- Is the command scoped narrowly enough?
- Is a timeout set?
- Is this mounted-drive recursion really necessary?
- Are dependent steps being run serially?
- Is there a cleanup step for every background process started?
- If interrupted, have residual processes been checked and cleaned?
