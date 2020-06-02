# Windows installer XML toolset (WiX) for Docker

It's currently difficult to create native high-quality installers for all major platforms in an automated way. The most difficult being Windows since the only native way to create Microsoft Installer (MSI) files is using Microsoft's own Windows-only tools.

The WiX project is Microsoft's oldest open-source project and fortunately it works on Linux under Wine using the .NET framework. It can generate MSI installers using `*.wxs` files.

Like any Wine-based software, installing it correctly is flaky and cumbersome. This project provides a reliable way for installing WiX using the magic that is Docker.

## How to use

You can play around with this Docker image by pulling it and starting a shell inside.

``` sh
docker run -i -t justmoon/wix /bin/bash
```

WiX is installed in the `/home/wix/wix` folder.

For production use I would recommend creating your own subimage using a Dockerfile like this:

``` bash
FROM justmoon/wix
MAINTAINER You <you@example.com>

ADD example.wxs /home/wix/example.wxs
RUN wine candle.exe /home/wix/example.wxs
RUN wine light.exe /home/wix/example.wixobj
```
