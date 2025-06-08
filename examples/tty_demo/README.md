# TTY Demo

This is a simple demo program that demonstrates how to use TTY with tinydbg.

## How to Run

1. First, create a new PTY pair using socat:
```bash
socat -d -d pty,raw,echo=0 pty,raw,echo=0
```

2. Note the two PTY paths from the output (e.g., `/dev/pts/23` and `/dev/pts/24`)

3. In one terminal, run the program with tinydbg using the first PTY:
```bash
tinydbg debug --tty /dev/pts/23 main.go
```

4. In another terminal, you can interact with the program in several ways:

   a. Using echo to write to the PTY:
   ```bash
   echo "hello" > /dev/pts/24
   ```

   b. Using cat to read from the PTY:
   ```bash
   cat /dev/pts/24
   ```

   c. Using socat to both read and write to the PTY (recommended):
   ```bash
   socat - /dev/pts/24
   ```

5. The program will:
   - Print a welcome message
   - Wait for your input
   - Echo back what you type
   - Continue until you type 'quit'

## Example Session

```
TTY Demo Program
Type something and press Enter (type 'quit' to exit):
> hello
You typed: hello
> world
You typed: world
> quit
Goodbye!
```

## Testing Tips

- Use `socat - /dev/pts/24` for the best interactive experience - it allows both reading and writing to the PTY
- You can use `echo` to write to the PTY
- You can use `cat` to read from the PTY
- Each PTY in the pair is connected - writing to one end will be read by the other 