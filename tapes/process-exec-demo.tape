Output tapes/process-exec-demo.gif
Output tapes/process-exec-demo.mp4

Require kernel

Set Shell "bash"
Set FontSize 32
Set Width 2400
Set Height 1200
Set TypingSpeed 100ms

Type "kernel browsers create -H -p process-exec-demo" Enter
Sleep 2s
Type "kernel browsers process exec process-exec-demo -- echo hello world!" Enter
Sleep 2s
Type "kernel browsers delete -y process-exec-demo" Enter
Sleep 5s
