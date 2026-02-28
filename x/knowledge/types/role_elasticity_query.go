package types

// QueryRoleElasticityRequest is the request for the RoleElasticity query (R29-3).
type QueryRoleElasticityRequest struct {
	Domain string `protobuf:"bytes,1,opt,name=domain,proto3" json:"domain,omitempty"`
}

func (m *QueryRoleElasticityRequest) Reset()         {}
func (m *QueryRoleElasticityRequest) String() string { return m.Domain }
func (m *QueryRoleElasticityRequest) ProtoMessage()  {}

func (m *QueryRoleElasticityRequest) GetDomain() string {
	if m != nil {
		return m.Domain
	}
	return ""
}

// QueryRoleElasticityResponse is the response for the RoleElasticity query (R29-3).
type QueryRoleElasticityResponse struct {
	Domain           string `protobuf:"bytes,1,opt,name=domain,proto3" json:"domain,omitempty"`
	AgentCorrect     uint64 `protobuf:"varint,2,opt,name=agent_correct,json=agentCorrect,proto3" json:"agent_correct,omitempty"`
	AgentIncorrect   uint64 `protobuf:"varint,3,opt,name=agent_incorrect,json=agentIncorrect,proto3" json:"agent_incorrect,omitempty"`
	HumanCorrect     uint64 `protobuf:"varint,4,opt,name=human_correct,json=humanCorrect,proto3" json:"human_correct,omitempty"`
	HumanIncorrect   uint64 `protobuf:"varint,5,opt,name=human_incorrect,json=humanIncorrect,proto3" json:"human_incorrect,omitempty"`
	AgentBonusBps    uint64 `protobuf:"varint,6,opt,name=agent_bonus_bps,json=agentBonusBps,proto3" json:"agent_bonus_bps,omitempty"`
	HumanBonusBps    uint64 `protobuf:"varint,7,opt,name=human_bonus_bps,json=humanBonusBps,proto3" json:"human_bonus_bps,omitempty"`
	AgentAccuracyBps uint64 `protobuf:"varint,8,opt,name=agent_accuracy_bps,json=agentAccuracyBps,proto3" json:"agent_accuracy_bps,omitempty"`
	HumanAccuracyBps uint64 `protobuf:"varint,9,opt,name=human_accuracy_bps,json=humanAccuracyBps,proto3" json:"human_accuracy_bps,omitempty"`
}

func (m *QueryRoleElasticityResponse) Reset()         {}
func (m *QueryRoleElasticityResponse) String() string { return m.Domain }
func (m *QueryRoleElasticityResponse) ProtoMessage()  {}
