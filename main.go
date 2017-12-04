package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

type word uint16

type tstack []word

func (cpu *tcpu) pushStack(v word) {
	cpu.stack = append(cpu.stack, v)
}

func (cpu *tcpu) popStack() word {
	l := len(cpu.stack)
	result := cpu.stack[l-1]
	cpu.stack = cpu.stack[:l-1]
	return result
}

type treg map[word]word

type tcpu struct {
	reg    treg
	memory *os.File
	stack  tstack
	input  *bufio.Reader
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: sinacore <filename>")
		os.Exit(0)
	}
	filename := os.Args[1]

	programm, err := os.OpenFile(filename, os.O_RDWR, 0755)
	if err != nil {
		fmt.Println("File " + filename + " not found")
		os.Exit(0)
	}

	memory, err := ioutil.TempFile("", "memory")
	if err != nil {
		panic(err)
	}
	defer os.Remove(memory.Name())
	io.Copy(memory, programm)

	reg := map[word]word{
		word(32768): word(0),
		word(32769): word(0),
		word(32770): word(0),
		word(32771): word(0),
		word(32772): word(0),
		word(32773): word(0),
		word(32774): word(0),
		word(32775): word(0),
	}

	cpu := tcpu{memory: memory, reg: reg, input: bufio.NewReader(os.Stdin)}
	cpu.jmp(0)
	for {
		number := cpu.getValue()
		switch number {
		case 0:
			cpu.halt()
		case 1:
			cpu.set(cpu.getRegNum(), cpu.getNumber())
		case 2:
			cpu.push(cpu.getNumber())
		case 3:
			cpu.pop(cpu.getRegNum())
		case 4:
			cpu.eq(cpu.getRegNum(), cpu.getNumber(), cpu.getNumber())
		case 5:
			cpu.gt(cpu.getRegNum(), cpu.getNumber(), cpu.getNumber())
		case 6:
			cpu.jmp(cpu.getNumber())
		case 7:
			cpu.jt(cpu.getNumber(), cpu.getNumber())
		case 8:
			cpu.jf(cpu.getNumber(), cpu.getNumber())
		case 9:
			cpu.add(cpu.getRegNum(), cpu.getNumber(), cpu.getNumber())
		case 10:
			cpu.mult(cpu.getRegNum(), cpu.getNumber(), cpu.getNumber())
		case 11:
			cpu.mod(cpu.getRegNum(), cpu.getNumber(), cpu.getNumber())
		case 12:
			cpu.and(cpu.getRegNum(), cpu.getNumber(), cpu.getNumber())
		case 13:
			cpu.or(cpu.getRegNum(), cpu.getNumber(), cpu.getNumber())
		case 14:
			cpu.not(cpu.getRegNum(), cpu.getNumber())
		case 15:
			cpu.rmem(cpu.getRegNum(), cpu.getNumber())
		case 16:
			cpu.wmem(cpu.getNumber(), cpu.getNumber())
		case 17:
			cpu.call(cpu.getNumber())
		case 18:
			cpu.ret()
		case 19:
			cpu.out(cpu.getNumber())
		case 20:
			cpu.in(cpu.getRegNum())
		case 21:
		}
	}
}

func (cpu *tcpu) in(a word) {
	var ch rune
	ch, _, err := cpu.input.ReadRune()
	if err != nil {
		panic(err)
	}
	cpu.reg[a] = word(ch)
}

func (cpu *tcpu) halt() {
	fmt.Println("Bye!")
	os.Exit(0)
}

func (cpu *tcpu) eq(a, b, c word) {
	if b == c {
		cpu.reg[a] = 1
	} else {
		cpu.reg[a] = 0
	}
}

func (cpu *tcpu) ret() {
	addr := cpu.popStack()
	cpu.jmp(addr)
}

func (cpu *tcpu) getCurrentPos() word {
	currentPos, err := cpu.memory.Seek(0, io.SeekCurrent)
	if err != nil {
		panic(err)
	}
	return word(currentPos) / 2
}

func (cpu *tcpu) call(a word) {
	cpu.pushStack(cpu.getCurrentPos())
	cpu.jmp(a)
}

func (cpu *tcpu) push(a word) {
	cpu.pushStack(a)
}

func (cpu *tcpu) pop(a word) {
	cpu.reg[a] = cpu.popStack()
}

func (cpu *tcpu) getLiteral() word {
	x := cpu.getValue()
	if x > 21 && int32(x) < 32768 {
		return x
	}
	panic("literal expected")
}

func (cpu *tcpu) getCmd() word {
	x := cpu.getValue()
	if x > 21 {
		panic("register address expected")
	}
	return x
}

func (cpu *tcpu) getRegNum() word {
	x := cpu.getValue()
	if int32(x) >= 32768 && int32(x) <= 32775 {
		return x
	}
	panic("register address expected")
}

func (cpu *tcpu) getNumber() word {
	x := cpu.getValue()
	if int32(x) > 32775 {
		panic("too large value")
	}
	if int32(x) >= 32768 && int32(x) <= 32775 {
		return cpu.reg[x]
	}
	return x
}

func (cpu *tcpu) getValue() word {
	var number word
	err := binary.Read(cpu.memory, binary.LittleEndian, &number)
	if err != nil {
		panic(err)
	}
	return number
}

func (cpu *tcpu) gt(a, b, c word) {
	if b > c {
		cpu.reg[a] = 1
	} else {
		cpu.reg[a] = 0
	}
}

func (cpu *tcpu) rmem(a, b word) {
	old := cpu.getCurrentPos()
	cpu.jmp(b)
	cpu.reg[a] = cpu.getNumber()
	cpu.jmp(old)
}

func (cpu *tcpu) wmem(a, b word) {
	old := cpu.getCurrentPos()
	cpu.jmp(a)
	err := binary.Write(cpu.memory, binary.LittleEndian, b)
	if err != nil {
		panic(err)
	}
	cpu.jmp(old)
}

func (cpu *tcpu) set(a, b word) {
	cpu.reg[a] = b
}

func (cpu *tcpu) jf(a, b word) {
	if a == 0 {
		cpu.jmp(b)
	}
}

func (cpu *tcpu) not(a, b word) {
	b = ^b
	b = b &^ 32768
	cpu.reg[a] = b
}

func (cpu *tcpu) jt(a, b word) {
	if a != 0 {
		cpu.jmp(b)
	}
}

func (cpu *tcpu) jmp(a word) {
	if _, err := cpu.memory.Seek(int64(2*a), io.SeekStart); err != nil {
		panic(err)
	}
}

func (cpu *tcpu) out(a word) {
	fmt.Print(string(a))
}

func (cpu *tcpu) mult(a, b, c word) {
	cpu.reg[a] = word((int32(b) * int32(c)) % 32768)
}

func (cpu *tcpu) mod(a, b, c word) {
	cpu.reg[a] = b % c
}

func (cpu *tcpu) or(a, b, c word) {
	cpu.reg[a] = b | c
}

func (cpu *tcpu) and(a, b, c word) {
	cpu.reg[a] = b & c
}

func (cpu *tcpu) add(a, b, c word) {
	cpu.reg[a] = word((int32(b) + int32(c)) % 32768)
}
