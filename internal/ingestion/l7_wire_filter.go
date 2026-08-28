package ingestion

import (
	"context"
	"sync/atomic"

	"ad-event-processor/internal/domain"
)

type L7WireFilter struct {
	secFetchEnabled         atomic.Bool
	clientHintsEnabled      atomic.Bool
	tlsALPNEnabled          atomic.Bool
	h2SettingsEnabled       atomic.Bool
	h2PseudoOrderEnabled    atomic.Bool
	h2DowngradeEnabled      atomic.Bool
	http1HeaderOrderEnabled atomic.Bool
	acceptEncodingEnabled   atomic.Bool
}

func NewL7WireFilter() *L7WireFilter {
	f := &L7WireFilter{}
	f.secFetchEnabled.Store(true)
	f.clientHintsEnabled.Store(true)
	f.tlsALPNEnabled.Store(true)
	f.h2SettingsEnabled.Store(true)
	f.h2PseudoOrderEnabled.Store(true)
	f.h2DowngradeEnabled.Store(true)
	f.http1HeaderOrderEnabled.Store(true)
	f.acceptEncodingEnabled.Store(true)
	return f
}

func (f *L7WireFilter) SetSecFetchEnabled(enabled bool) {
	f.secFetchEnabled.Store(enabled)
}

func (f *L7WireFilter) SetClientHintsPlatformEnabled(enabled bool) {
	f.clientHintsEnabled.Store(enabled)
}

func (f *L7WireFilter) SetTLSALPNMismatchEnabled(enabled bool) {
	f.tlsALPNEnabled.Store(enabled)
}

func (f *L7WireFilter) SetH2SettingsEnabled(enabled bool) {
	f.h2SettingsEnabled.Store(enabled)
}

func (f *L7WireFilter) SetH2PseudoOrderEnabled(enabled bool) {
	f.h2PseudoOrderEnabled.Store(enabled)
}

func (f *L7WireFilter) SetH2DowngradeEnabled(enabled bool) {
	f.h2DowngradeEnabled.Store(enabled)
}

func (f *L7WireFilter) SetHTTP1HeaderOrderEnabled(enabled bool) {
	f.http1HeaderOrderEnabled.Store(enabled)
}

func (f *L7WireFilter) SetAcceptEncodingEnabled(enabled bool) {
	f.acceptEncodingEnabled.Store(enabled)
}

func (f *L7WireFilter) Check(ctx context.Context, evt *domain.Event) error {
	if f == nil || evt == nil {
		return nil
	}
	if f.secFetchEnabled.Load() && secFetchAnomaly(evt.UA, evt.SecFetchPresent, evt.SecFetchMode, evt.SecFetchDest) {
		addFraudSignal(evt, FraudReasonSecFetchAnomaly)
	}
	if f.clientHintsEnabled.Load() && clientHintsPlatformMismatch(evt.UA, evt.SecCHUAPlatform, evt.SecCHUAMobile) {
		addFraudSignal(evt, FraudReasonClientHintsMismatch)
	}
	if f.tlsALPNEnabled.Load() && tlsALPNBrowserMismatch(evt.UA, evt.TLSALPN) {
		addFraudSignal(evt, FraudReasonTLSALPNMismatch)
	}
	if evt.IngressH2 != 0 {
		if f.h2SettingsEnabled.Load() && h2SettingsAnomaly(evt.UA, evt.H2WireFlags, evt.H2EnablePush, evt.H2InitialWindow, evt.H2WindowUpdateInc) {
			addFraudSignal(evt, FraudReasonH2SettingsMismatch)
		}
		if f.h2PseudoOrderEnabled.Load() && h2PseudoOrderMismatch(evt.UA, evt.H2PseudoOrder, evt.H2PseudoOrderCount) {
			addFraudSignal(evt, FraudReasonH2PseudoOrder)
		}
		if f.h2DowngradeEnabled.Load() && h2DowngradeArtifact(evt.H2WireFlags) {
			addFraudSignal(evt, FraudReasonH2DowngradeArtifact)
		}
	} else if f.http1HeaderOrderEnabled.Load() && http1HeaderOrderMismatch(evt.UA, evt.HTTP1HeaderOrder[:], evt.HTTP1HeaderOrderCount, evt.SecFetchPresent) {
		addFraudSignal(evt, FraudReasonHeaderOrderMismatch)
	}
	if f.acceptEncodingEnabled.Load() && acceptEncodingBrowserMismatch(evt.UA, evt.AcceptEncodingFlags, evt.AcceptEncodingSet) {
		addFraudSignal(evt, FraudReasonAcceptEncodingMismatch)
	}
	return nil
}
