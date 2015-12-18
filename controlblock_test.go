package dmr

import "testing"

func testCSBK(want *ControlBlock, t *testing.T) *ControlBlock {
	want.SrcID = 2042214
	want.DstID = 2043044

	data, err := want.Bytes()
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	test, err := ParseControlBlock(data)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if test.SrcID != want.SrcID || test.DstID != want.DstID {
		t.Fatal("decode failed, ID wrong")
	}

	return test
}

func TestCSBKOutboundActivation(t *testing.T) {
	want := &ControlBlock{
		Opcode: OutboundActivationOpcode,
		Data:   &OutboundActivation{},
	}
	test := testCSBK(want, t)

	_, ok := test.Data.(*OutboundActivation)
	switch {
	case !ok:
		t.Fatalf("decode failed: expected UnitToUnitVoiceServiceRequest, got %T", test.Data)

	default:
		t.Logf("decode: %s", test.String())
	}
}

func TestCSBKUnitToUnitVoiceServiceRequest(t *testing.T) {
	want := &ControlBlock{
		Opcode: UnitToUnitVoiceServiceRequestOpcode,
		Data: &UnitToUnitVoiceServiceRequest{
			Options: 0x2a,
		},
	}
	test := testCSBK(want, t)

	d, ok := test.Data.(*UnitToUnitVoiceServiceRequest)
	switch {
	case !ok:
		t.Fatalf("decode failed: expected UnitToUnitVoiceServiceRequest, got %T", test.Data)

	case d.Options != 0x2a:
		t.Fatalf("decode failed, options wrong")

	default:
		t.Logf("decode: %s", test.String())
	}
}

func TestCSBKUnitToUnitVoiceServiceAnswerResponse(t *testing.T) {
	want := &ControlBlock{
		Opcode: UnitToUnitVoiceServiceAnswerResponseOpcode,
		Data: &UnitToUnitVoiceServiceAnswerResponse{
			Options:  0x17,
			Response: 0x2a,
		},
	}
	test := testCSBK(want, t)

	d, ok := test.Data.(*UnitToUnitVoiceServiceAnswerResponse)
	switch {
	case !ok:
		t.Fatalf("decode failed: expected UnitToUnitVoiceServiceAnswerResponse, got %T", test.Data)

	case d.Response != 0x2a:
		t.Fatalf("decode failed, response wrong")

	case d.Options != 0x17:
		t.Fatalf("decode failed, options wrong")

	default:
		t.Logf("decode: %s", test.String())
	}
}

func TestCSBKNegativeAcknowledgeResponse(t *testing.T) {
	want := &ControlBlock{
		Opcode: NegativeAcknowledgeResponseOpcode,
		Data: &NegativeAcknowledgeResponse{
			ServiceType: 0x01,
			Reason:      0x02,
		},
	}
	test := testCSBK(want, t)

	d, ok := test.Data.(*NegativeAcknowledgeResponse)
	switch {
	case !ok:
		t.Fatalf("decode failed: expected NegativeAcknowledgeResponse, got %T", test.Data)

	case d.ServiceType != 0x01:
		t.Fatalf("decode failed, service type wrong")

	case d.Reason != 0x02:
		t.Fatalf("decode failed, reason wrong")

	default:
		t.Logf("decode: %s", test.String())
	}
}

func TestCSBKPreamble(t *testing.T) {
	want := &ControlBlock{
		Opcode: PreambleOpcode,
		Data: &Preamble{
			DataFollows: true,
			DstIsGroup:  false,
			Blocks:      0x10,
		},
	}
	test := testCSBK(want, t)

	d, ok := test.Data.(*Preamble)
	switch {
	case !ok:
		t.Fatalf("decode failed: expected Preamble, got %T", test.Data)

	case !d.DataFollows:
		t.Fatalf("decode failed, data follows wrong")

	case d.DstIsGroup:
		t.Fatalf("decode failed, dst is group wrong")

	case d.Blocks != 0x10:
		t.Fatalf("decode failed, blocks wrong")

	default:
		t.Logf("decode: %s", test.String())
	}
}
