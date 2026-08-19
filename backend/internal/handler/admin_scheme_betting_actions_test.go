package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"caipiao/backend/internal/auth"
	"caipiao/backend/internal/middleware"
	"caipiao/backend/internal/schemebetting"
)

type fakeSchemeBettingActions struct {
	rearmSchemeID string
	rearmActor    string
	rearmReason   string
	cancelID      int64
	resolveID     int64
	resolveActor  string
	resolveReason string
	resolution    schemebetting.UnknownResolution
}

func (fake *fakeSchemeBettingActions) EnableEventScheme(_ context.Context, schemeID, actor, reason string) error {
	return fake.RearmEventScheme(context.Background(), schemeID, actor, reason)
}

func (fake *fakeSchemeBettingActions) RearmEventScheme(_ context.Context, schemeID, actor, reason string) error {
	fake.rearmSchemeID = schemeID
	fake.rearmActor = actor
	fake.rearmReason = reason
	return nil
}

func (fake *fakeSchemeBettingActions) CancelEventBet(_ context.Context, outboxID int64, _, _ string) error {
	fake.cancelID = outboxID
	return nil
}

func (fake *fakeSchemeBettingActions) ResolveUnknownEventBet(_ context.Context, outboxID int64, actor, reason string, resolution schemebetting.UnknownResolution) error {
	fake.resolveID = outboxID
	fake.resolveActor = actor
	fake.resolveReason = reason
	fake.resolution = resolution
	return nil
}

func TestAdminSchemeBettingRearmRequiresReasonAndCapturesActor(t *testing.T) {
	fake := &fakeSchemeBettingActions{}
	h := &Handler{schemeBettingActions: fake}
	shortRequest := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"reason":"bad"}`))
	shortRequest = shortRequest.WithContext(middleware.WithClaims(shortRequest.Context(), adminClaims("ops-admin")))
	shortRequest.SetPathValue("schemeId", "scheme-1")
	shortResponse := httptest.NewRecorder()
	h.AdminSchemeBettingRearm(shortResponse, shortRequest)
	if shortResponse.Code != http.StatusOK || !strings.Contains(shortResponse.Body.String(), `"code":42200`) {
		t.Fatalf("short reason response = %d %s", shortResponse.Code, shortResponse.Body.String())
	}
	if fake.rearmSchemeID != "" {
		t.Fatal("service called for an invalid reason")
	}

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"reason":"manual recovery"}`))
	request = request.WithContext(middleware.WithClaims(request.Context(), adminClaims("ops-admin")))
	request.SetPathValue("schemeId", "scheme-1")
	response := httptest.NewRecorder()
	h.AdminSchemeBettingRearm(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"rearmed":true`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
	if fake.rearmSchemeID != "scheme-1" || fake.rearmActor != "ops-admin" || fake.rearmReason != "manual recovery" {
		t.Fatalf("service args = %#v", fake)
	}
}

func TestAdminSchemeBettingCancelRejectsInvalidOutboxID(t *testing.T) {
	fake := &fakeSchemeBettingActions{}
	h := &Handler{schemeBettingActions: fake}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"reason":"manual cancel"}`))
	request.SetPathValue("outboxId", "invalid")
	response := httptest.NewRecorder()
	h.AdminSchemeBettingCancel(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"code":42200`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
	if fake.cancelID != 0 {
		t.Fatal("service called for an invalid outbox id")
	}
}

func TestAdminSchemeBettingActionsRequireSuperAdminRole(t *testing.T) {
	fake := &fakeSchemeBettingActions{}
	h := &Handler{schemeBettingActions: fake}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"reason":"manual recovery"}`))
	claims := adminClaims("ops-admin")
	claims.AdminRoleID = "r_fin_approve"
	request = request.WithContext(middleware.WithClaims(request.Context(), claims))
	request.SetPathValue("schemeId", "scheme-1")
	response := httptest.NewRecorder()
	h.AdminSchemeBettingRearm(response, request)
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), `"code":40300`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
	if fake.rearmSchemeID != "" {
		t.Fatal("non-super role called scheme betting action")
	}
}

func TestAdminSchemeBettingResolveUnknownCapturesEvidenceAndActor(t *testing.T) {
	fake := &fakeSchemeBettingActions{}
	h := &Handler{schemeBettingActions: fake}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{
		"reason":"manual provider reconciliation",
		"outcome":"accepted",
		"evidence":"provider order detail checked",
		"providerOrderId":"provider-1",
		"acceptedPeriod":"20260819001",
		"providerAmount":12.5,
		"providerAccountId":9,
		"providerCurrency":"CNY"
	}`))
	request = request.WithContext(middleware.WithClaims(request.Context(), adminClaims("ops-admin")))
	request.SetPathValue("outboxId", "42")
	response := httptest.NewRecorder()
	h.AdminSchemeBettingResolveUnknown(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"resolved":true`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
	if fake.resolveID != 42 || fake.resolveActor != "ops-admin" || fake.resolveReason != "manual provider reconciliation" {
		t.Fatalf("service args = %#v", fake)
	}
	if fake.resolution.ProviderOrderID != "provider-1" || fake.resolution.AcceptedPeriod != "20260819001" ||
		fake.resolution.ProviderAmount != 12.5 || fake.resolution.ProviderAccountID != 9 || fake.resolution.ProviderCurrency != "CNY" {
		t.Fatalf("resolution = %#v", fake.resolution)
	}
}

func adminClaims(subject string) auth.Claims {
	return auth.Claims{AdminRoleID: "r_super", RegisteredClaims: jwt.RegisteredClaims{Subject: subject}}
}
