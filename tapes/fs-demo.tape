Output tapes/fs-demo.gif
Output tapes/fs-demo.mp4

Require kernel

Set Shell "bash"
Set FontSize 32
Set Width 2400
Set Height 1200
Set TypingSpeed 20ms

Type "kernel browsers create -H -p demo > /dev/null" Enter
Sleep 2s
Type "kernel browsers fs new-directory demo --path /tmp/test" Enter
Sleep 2s
Type "echo 'hello world' > /tmp/hello_world.txt" Enter
Sleep 2s
Type "kernel browsers fs write-file demo --source /tmp/hello_world.txt --path /tmp/test/hello_world" Enter
Sleep 2s
Type "kernel browsers fs list-files demo --path /tmp/test" Enter
Sleep 2s
Type "kernel browsers fs file-info demo --path /tmp/test/hello_world" Enter
Sleep 2s
Type "kernel browsers fs read-file demo --path /tmp/test/hello_world" Enter
Sleep 2s
Type "kernel browsers fs download-dir-zip demo --path /tmp/test --output /tmp/test.zip" Enter
Sleep 2s
Type "unzip -l /tmp/test.zip" Enter
Sleep 2s
Type "kernel browsers delete -y demo" Enter
Sleep 5s
