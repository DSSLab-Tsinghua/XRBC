package message

type TypeOfMessage int

const (
	RBC_ALL             TypeOfMessage = 1
	RBC_SEND            TypeOfMessage = 2
	RBC_ECHO            TypeOfMessage = 3
	RBC_READY           TypeOfMessage = 4
	ABA_ALL             TypeOfMessage = 5
	ABA_BVAL            TypeOfMessage = 6
	ABA_AUX             TypeOfMessage = 7
	ABA_CONF            TypeOfMessage = 8
	ABA_FINAL           TypeOfMessage = 9
	PRF                 TypeOfMessage = 10
	ECRBC_ALL           TypeOfMessage = 11
	CBC_ALL             TypeOfMessage = 12
	CBC_SEND            TypeOfMessage = 13
	CBC_REPLY           TypeOfMessage = 14
	CBC_ECHO            TypeOfMessage = 15
	CBC_EREPLY          TypeOfMessage = 16
	CBC_READY           TypeOfMessage = 17
	MVBA_DISTRIBUTE     TypeOfMessage = 18
	EVCBC_ALL           TypeOfMessage = 19
	RETRIEVE            TypeOfMessage = 20
	SIMPLE_SEND         TypeOfMessage = 21
	ECHO_SEND           TypeOfMessage = 22
	ECHO_REPLY          TypeOfMessage = 23
	GC_ALL              TypeOfMessage = 24
	GC_FORWARD          TypeOfMessage = 25
	GC_ECHO             TypeOfMessage = 26
	GC_READY            TypeOfMessage = 27
	GC_CAST             TypeOfMessage = 28
	SIMPLE_PROOF        TypeOfMessage = 29
	PLAINCBC_ALL        TypeOfMessage = 30
	DISPERSE            TypeOfMessage = 31
	MBA_ALL             TypeOfMessage = 32
	MBA_ECHO            TypeOfMessage = 33
	MBA_FORWARD         TypeOfMessage = 34
	MBA_DISTRIBUTE      TypeOfMessage = 35
	PBFTEC_PP           TypeOfMessage = 36
	HotStuff_Msg        TypeOfMessage = 37
	CrossGroup_CROSS    TypeOfMessage = 38
	CrossGroup_REP      TypeOfMessage = 39
	CrossGroup_CONFIRM  TypeOfMessage = 40
	CrossGroup_FETCH    TypeOfMessage = 41
	CrossGroup_CATCHUP  TypeOfMessage = 42
	CrossAdd_Disperse   TypeOfMessage = 43
	CrossAdd_Reconsruct TypeOfMessage = 44
	CrossAdd_Cross      TypeOfMessage = 45
	CrossAdd_Share      TypeOfMessage = 46
	CrossDelta_Vote     TypeOfMessage = 47
)

type ProtocolType int

const (
	RBC           ProtocolType = 1
	ABA           ProtocolType = 2
	ECRBC         ProtocolType = 3
	CBC           ProtocolType = 4
	EVCBC         ProtocolType = 5
	MVBA          ProtocolType = 6
	SIMPLE        ProtocolType = 7
	ECHO          ProtocolType = 8
	GC            ProtocolType = 9
	SIMPLEPROOF   ProtocolType = 10
	PLAINCBC      ProtocolType = 11
	MBA           ProtocolType = 12
	PBFTEC        ProtocolType = 13
	CrossRA       ProtocolType = 14
	CrossABC      ProtocolType = 15
	CrossSig      ProtocolType = 16
	CrossSigGroup ProtocolType = 18
	CrossADD      ProtocolType = 19
	CrossADDC     ProtocolType = 20
	HotStuff      ProtocolType = 17
)

type HashMode int

const (
	DEFAULT_HASH HashMode = 0
	MERKLE       HashMode = 1
)
