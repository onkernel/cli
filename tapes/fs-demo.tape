Output tapes/fs-demo.gif
Output tapes/fs-demo.mp4

Require kernel

Set Shell "bash"
Set FontSize 32
Set Width 2400
Set Height 1200
Set TypingSpeed 100ms

Type "kernel browsers create -H -p fs-demo" Enter
Sleep 2s
Type "kernel browsers fs new-directory fs-demo --path /tmp/test" Enter
Sleep 2s
Type "echo 'hello world' > /tmp/hello_world.txt" Enter
Sleep 2s
Type "kernel browsers fs write-file fs-demo --source /tmp/hello_world.txt --path /tmp/test/hello_world" Enter
Sleep 2s
Type "kernel browsers fs list-files fs-demo --path /tmp/test" Enter
Sleep 2s
Type "kernel browsers fs file-info fs-demo --path /tmp/test/hello_world" Enter
Sleep 2s
Type "kernel browsers fs read-file fs-demo --path /tmp/test/hello_world" Enter
Sleep 2s
Type "kernel browsers fs download-dir-zip fs-demo --path /tmp/test --output /tmp/test.zip" Enter
Sleep 2s
Type "unzip -l /tmp/test.zip" Enter
Sleep 2s
Type "kernel browsers delete -y fs-demo" Enter
Sleep 5s
