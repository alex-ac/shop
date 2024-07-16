# Shop

A [CIPD](https://chromium.googlesource.com/infra/luci/luci-go/+/main/cipd/README.md)
inspired package registry/deployer.

Shop is a package registry/deployment system for development tools.

Just like in CIPD A package has a name and list of content-addressed instances,
where slashes in package names form a hierarchy of packages and an instance
is a zip file with the package contents.

Shop is different from apt, brew, pip, npm, etc. in that it is not tied to
a specific OS or language. Shop is different from CIPD in that it is not tied
to chromium project or google infrastructure.

## Versions

A package instance can be referenced by a tuple (package name, version).
A version is one of:

 * A hash of the instance contents, e.g. `bec8e88201949be06b06174178c2f62b81e4008e`.
    This is also called instance id.
 * A key-value tag: e.g. `git_revision:deadbeef`, if it's unique among all
    instances of the package.
 * A ref, e.g. `latest`.

## Tags

A package instance can be marked with tags, where a tag is a colon-separated
key-value pair, e.g. `git_revision:deadbeef`. If some tag points to only one
instance, such tag can be used as version identifier.

## Refs

A package can have a git-like refs, where a ref of package points to one of
the instances of the package by id.

## Platforms

If a package is platform-specific, the package name should have a `/os-arch`
suffix.

## Access-Control

Unlike CIPD, Shop does not have a fine-granular builtin access-control mechanism.
If you have access to registry, you have same access to all packages inside that
registry. Shop supports working with multiple registries and allows granting
different access to different registries.

In most common use-case, users only need to have read-only access to registry.
And only registry owners need read-write access to upload new packages. With
that in mind, it's not really a hard requirement to have an ACL support.

Shop aims to have no backend service and allows client to talk directly to the
storage. With that in mind it's really difficult to built any sort of ACL,
unless storage provides some kind of ACL on it's own.

## Why?

I like writing C++ code. And I want modern llvm toolchain/SDK to do that. And
the only good way to have them for some reason - building them myself. And
it's too large to build from scratch for any ocasion. Aim is to create a
convenient storage for that kind of tools and a set of automated scripts to
build and automatically publish new versions.

## Backend

Shop is made to use any S3 compatible storage as a backend. All the API
endpoints are saved as JSON objects into the storage so any client with read
only access can have the data. Updating those objects is a bit cumbersome but
could be easily done automatically. Upside of this approach is: you don't need
to have a client to access registry and download packgaes if you have access
to the storage. You can even bootstrap the client from that same registry.

## Official registry & packages

It would be weird to create a registry and not to run it, I intend to create
a set of scripts to build & publish packages for all the tools I consider
useful. And I will publish most of them into the public registry. However, due
to some licensing issues with some of the packages (some software is illegal to
redistribute especially in a modified form), some of the packages would only be
uploaded to my private registry. You're free to use the scripts to build your
own.
