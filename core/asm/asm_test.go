package asm

import (
	"fadingrose/rosy-nigh/core/vm"
	"reflect"
	"testing"
)

func TestDisAssembler(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []vm.Instruction
	}{
		{
			name:  "Basic Instructions",
			input: []byte{0x01, 0x02, 0x03},
			expected: []vm.Instruction{
				{OpCode: 0x01, PC: 0},
				{OpCode: 0x02, PC: 1},
				{OpCode: 0x03, PC: 2},
			},
		},
		{
			name:  "Push Instructions",
			input: []byte{0x60, 0x01, 0x61, 0x01, 0x02, 0x62, 0x01, 0x02, 0x03},
			expected: []vm.Instruction{
				{OpCode: 0x60, PC: 0, Operand: []byte{0x01}},
				{OpCode: 0x61, PC: 2, Operand: []byte{0x01, 0x02}},
				{OpCode: 0x62, PC: 5, Operand: []byte{0x01, 0x02, 0x03}},
			},
		},
		{
			name:     "Empty Input",
			input:    []byte{},
			expected: []vm.Instruction{},
		},
		{
			name:  "Invalid Push Instruction",
			input: []byte{0x60, 0x01, 0x61},
			expected: []vm.Instruction{
				{OpCode: 0x60, PC: 0, Operand: []byte{0x01}},
				{OpCode: 0x61, PC: 2, Operand: []byte{0x00, 0x00}},
			},
		},
		{
			name:  "Invalid Input",
			input: append([]byte{0x60, 0x01}, make([]byte, 3)...),
			expected: []vm.Instruction{
				{OpCode: 0x60, PC: 0, Operand: []byte{0x01}},
				{OpCode: 0x00, PC: 2},
				{OpCode: 0x00, PC: 3},
				{OpCode: 0x00, PC: 4},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DisAssembler(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("DisAssembler(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestAssembler(t *testing.T) {
	tests := []struct {
		name     string
		input    []vm.Instruction
		expected []byte
	}{
		{
			name: "Basic Instructions",
			input: []vm.Instruction{
				{OpCode: 0x01},
				{OpCode: 0x02},
				{OpCode: 0x03},
			},
			expected: []byte{0x01, 0x02, 0x03},
		},
		{
			name: "Push Instructions",
			input: []vm.Instruction{
				{OpCode: 0x60, Operand: []byte{0x01}},
				{OpCode: 0x61, Operand: []byte{0x01, 0x02}},
				{OpCode: 0x62, Operand: []byte{0x01, 0x02, 0x03}},
			},
			expected: []byte{0x60, 0x01, 0x61, 0x01, 0x02, 0x62, 0x01, 0x02, 0x03},
		},
		{
			name:     "Empty Input",
			input:    []vm.Instruction{},
			expected: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Assembler(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Assembler(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "Basic Instructions",
			input: []byte{0x01, 0x02, 0x03},
		},
		{
			name:  "Push Instructions",
			input: []byte{0x60, 0x01, 0x61, 0x01, 0x02, 0x62, 0x01, 0x02, 0x03},
		},
		{
			name:  "Empty Input",
			input: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			disassembled := DisAssembler(tt.input)
			reassembled := Assembler(disassembled)
			if !reflect.DeepEqual(reassembled, tt.input) {
				t.Errorf("RoundTrip failed: input %v, reassembled %v", tt.input, reassembled)
			}
		})
	}
}

func FuzzAssemblerDisAssembler(f *testing.F) {
	// Seed the fuzzer with some initial data
	f.Add([]byte{0x01, 0x02, 0x03})
	f.Add([]byte{0x04, 0x05})
	f.Add([]byte{0x06})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Disassemble the byte slice
		instructions := DisAssembler(data)

		// Reassemble the instructions back to a byte slice
		reassembled := Assembler(instructions)

		// Ensure that the reassembled byte slice matches the original data
		if len(reassembled) != len(data) {
			t.Errorf("Reassembled byte slice length mismatch: got %d, want %d", len(reassembled), len(data))
		}
		for i := range data {
			if reassembled[i] != data[i] {
				t.Errorf("Byte mismatch at index %d: got %x, want %x", i, reassembled[i], data[i])
			}
		}
	})
}
