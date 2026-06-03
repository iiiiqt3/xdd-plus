package baseutils

// pb10字段xor
func SaePb10T(pb10 []uint8, pb9 []uint8) []uint8 {
	result := make([]uint8, len(pb10))
	for i := uint32(0); i < 9; i++ {
		for j := uint32(0); j < 4; j++ {
			for k := uint32(0); k < 4; k++ {
				for m := uint32(0); m < 0x100; m++ {
					for l := uint32(0); l < 4; l++ {
						index := (i << 14) + (j << 12) + (k << 10) + (m << 2)
						w10 := i*j*k + m + l
						w11 := w10 + 7
						w10 = w10 + 0x86
						if w11 >= 0 {
							w10 = w11
						}
						w10 = w10 & 0xffffff80
						w10 = w11 - w10
						result[l+index] = pb9[w10] ^ pb10[index+l]
					}
				}
			}
		}
	}
	return result
}

// pb12字段xor
func SaePb12T(pb12 []uint8, pb11 []uint8) []uint8 {
	result := make([]uint8, len(pb12))
	for i := uint32(0); i < 9; i++ {
		for j := uint32(0); j < 4; j++ {
			for k := uint32(0); k < 4; k++ {
				for l := uint32(0); l < 3; l++ {
					for m := uint32(0); m < 2; m++ {
						for n := uint32(0); n < 0x80; n++ {
							index := i*0x3000 + j*0xc00 + k*0x300 + (l << 8) + (m << 7)
							w10 := i*j*k + m*l + n
							w11 := w10 + 0x17
							w10 = w10 + 0x56
							if w11 >= 0 {
								w10 = w11
							}
							w10 = w10 & 0xffffffc0
							w10 = w11 - w10
							result[index+n] = pb11[w10] ^ pb12[index+n]
						}
					}
				}
			}
		}
	}
	return result
}
