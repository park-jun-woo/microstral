// https://parkjunwoo.com/microstral/pkg/pack/pack.go
package pack

import "fmt"

// SetBitUint8 sets len bits starting at position 'index' (0=MSB) in target to the given value.
// For example, if target = 0x00, index = 0, len = 3, value = 0b101, then bits 7,6,5 will be set to 101.
// Returns the modified target or an error if index/len are invalid or value does not fit in len bits.
func SetBitUint8(target uint8, index uint8, length uint8, value uint8) (uint8, error) {
	// 유효성 검사: length는 0이 아니고, index < 8, 그리고 index+length가 8 이하인지 확인
	if length == 0 || index >= 8 || index+length > 8 {
		return 0, fmt.Errorf("invalid index (%d) or length (%d), must satisfy: length > 0, index < 8 and index+length <= 8", index, length)
	}
	// value가 length 비트 내에 표현 가능한지 확인
	if value >= (1 << length) {
		return 0, fmt.Errorf("value %d does not fit in %d bits", value, length)
	}

	// 빅엔디안 기준에서 index 0은 최상위 비트(비트 위치 7)
	// 수정할 비트 블록은 시작 위치(start) = 7 - index, 그리고 길이 length이므로,
	// 블록의 최하위 비트 위치는 shift = 7 - index - (length - 1)
	shift := 7 - index - length + 1
	// 해당 블록의 마스크 계산: 예) length=3이면, mask = (1<<3)-1 = 0b111, 그리고 shift만큼 왼쪽 이동
	mask := ((1 << length) - 1) << shift

	// 기존 target에서 해당 비트 블록을 클리어하고, value를 해당 위치로 설정
	result := (target &^ uint8(mask)) | ((value & ((1 << length) - 1)) << shift)
	return result, nil
}

// GetBitUint8 extracts len bits starting at position 'index' (0=MSB) from target.
// Returns the extracted bits as the lower len bits of the result.
// For example, if target = 0xA0 (10100000 in binary), index = 0, len = 3, then the result is 0b101 (i.e. 5).
func GetBitUint8(target uint8, index uint8, length uint8) (uint8, error) {
	if length == 0 || index >= 8 || index+length > 8 {
		return 0, fmt.Errorf("invalid index (%d) or length (%d), must satisfy: length > 0, index < 8 and index+length <= 8", index, length)
	}

	shift := 7 - index - length + 1
	mask := (1 << length) - 1
	result := (target >> shift) & uint8(mask)
	return result, nil
}
