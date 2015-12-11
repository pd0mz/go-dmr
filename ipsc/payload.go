package ipsc

func SwapPayloadBytes(payload []byte) {
	for i := 0; i < len(payload)-1; i += 2 {
		payload[i], payload[i+1] = payload[i+1], payload[i]
	}
}
