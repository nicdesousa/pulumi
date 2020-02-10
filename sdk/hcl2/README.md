# Pulumi HCL2 SDK

This directory contains support for writing Pulumi programs in Pulumi-flavored HCL2. There are two aspects to this:

* `pulumi/` contains the HCL2 execution engine;
* `pulumi-language-hcl/` contains the language host plugin that the Pulumi engine uses to orchestrate updates.

To author a Pulumi program in HCL2, simply say so in your `Pulumi.yaml`

    name: <my-project>
    runtime: hcl2

and ensure you have `pulumi-language-hcl2` on your path (it is distributed in the Pulumi download automatically).

By default, the language plugin will use your project's name, `<my-project>`, as the executable that it loads. This too
must be on your path for the language provider to load it when you run `pulumi preview` or `pulumi up`.
