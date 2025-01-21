
rem cd capnp
rem capnp compile -I.. -ogo:../gen/go/persistence persistent.capnp
rem cd ..

capnp compile -I. -ogo:. storage.capnp spawner.capnp webview.capnp

