package audio

import (
	"fmt"
	"strings"

	"github.com/jfreymuth/pulse"
	"github.com/jfreymuth/pulse/proto"
)

var pulseClient *pulse.Client

func getClient() (*pulse.Client, error) {
	if pulseClient != nil {
		return pulseClient, nil
	}
	c, err := pulse.NewClient()
	if err != nil {
		return nil, err
	}
	pulseClient = c
	return c, nil
}

// CreateVirtualMic comunica com a PulseAudio RPC Daemon sem usar pactl cli.
func CreateVirtualMic(sinkName string, description string) error {
	DestroyVirtualMic(sinkName) 

	c, err := getClient()
	if err != nil {
		return fmt.Errorf("daemon inacessível: %v", err)
	}

	var reply proto.LoadModuleReply
	err = c.RawRequest(&proto.LoadModule{
		Name: "module-null-sink",
		Args: fmt.Sprintf("sink_name=%s sink_properties=device.description=\"%s_Internal\"", sinkName, description),
	}, &reply)

	if err != nil {
		return fmt.Errorf("LoadModule RPC Remote Call falhou (null-sink): %v", err)
	}

	// Remapeia o "Monitor" do null-sink para um Microphone Source real
	var reply2 proto.LoadModuleReply
	err = c.RawRequest(&proto.LoadModule{
		Name: "module-remap-source",
		Args: fmt.Sprintf("source_name=%s_mic master=%s.monitor source_properties=\"device.description='%s' device.class=sound\"", sinkName, sinkName, description),
	}, &reply2)

	if err != nil {
		return fmt.Errorf("LoadModule RPC Remote Call falhou (remap-source): %v", err)
	}

	return nil
}

// DestroyVirtualMic lista e remove modulos limpando memória de áudio invisivelmente
func DestroyVirtualMic(sinkName string) {
	c, err := getClient()
	if err != nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			// Ignora pânicos do client caso a ligação socket dbus esteja morta
		}
	}()

	var reply proto.GetModuleInfoListReply
	err = c.RawRequest(&proto.GetModuleInfoList{}, &reply)
	if err != nil {
		return
	}

	for _, mod := range reply {
		if (mod.ModuleName == "module-null-sink" || mod.ModuleName == "module-remap-source") && strings.Contains(mod.ModuleArgs, sinkName) {
			c.RawRequest(&proto.UnloadModule{ModuleIndex: mod.ModuleIndex}, nil)
		}
	}
}
