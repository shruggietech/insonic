// SPDX-License-Identifier: Apache-2.0
package secrets

import (
	"context"
	"github.com/godbus/dbus/v5"
	"github.com/shruggietech/insonic/internal/contracts"
	"time"
)

const serviceName = "org.freedesktop.secrets"
const servicePath = dbus.ObjectPath("/org/freedesktop/secrets")
const collectionPath = dbus.ObjectPath("/org/freedesktop/secrets/aliases/default")
const serviceInterface = "org.freedesktop.Secret.Service"
const collectionInterface = "org.freedesktop.Secret.Collection"
const itemInterface = "org.freedesktop.Secret.Item"

type linuxNative struct{ service string }
type dbusSecret struct {
	Session     dbus.ObjectPath
	Parameters  []byte
	Value       []byte
	ContentType string
}

func newNative(service string) nativeBackend { return linuxNative{service} }
func sessionBus(ctx context.Context) (*dbus.Conn, error) {
	bound, cancel := context.WithTimeout(ctx, 5*time.Second)
	conn, e := dbus.SessionBusPrivateNoAutoStartup(dbus.WithContext(bound))
	if e != nil {
		cancel()
		return nil, contracts.Fail("unavailable")
	}
	context.AfterFunc(conn.Context(), cancel)
	if e = conn.Auth(nil); e == nil {
		e = conn.Hello()
	}
	if e != nil {
		conn.Close()
		return nil, contracts.Fail("unavailable")
	}
	return conn, nil
}
func unlocked(ctx context.Context, conn *dbus.Conn, path dbus.ObjectPath, iface string) error {
	var value dbus.Variant
	e := conn.Object(serviceName, path).CallWithContext(ctx, "org.freedesktop.DBus.Properties.Get", 0, iface, "Locked").Store(&value)
	if e != nil {
		return contracts.Fail("unavailable")
	}
	locked, ok := value.Value().(bool)
	if !ok || locked {
		return contracts.Fail("unavailable")
	}
	return nil
}
func (n linuxNative) available(ctx context.Context) error {
	conn, e := sessionBus(ctx)
	if e != nil {
		return e
	}
	defer conn.Close()
	return unlocked(ctx, conn, collectionPath, collectionInterface)
}
func (n linuxNative) find(ctx context.Context, conn *dbus.Conn, id string) (dbus.ObjectPath, error) {
	var available, locked []dbus.ObjectPath
	e := conn.Object(serviceName, servicePath).CallWithContext(ctx, serviceInterface+".SearchItems", 0, map[string]string{"service": n.service, "username": id}).Store(&available, &locked)
	if e != nil || len(locked) > 0 || len(available) > 1 {
		return "", contracts.Fail("unavailable")
	}
	if len(available) == 0 {
		return "", contracts.Fail("not_found")
	}
	if e = unlocked(ctx, conn, available[0], itemInterface); e != nil {
		return "", e
	}
	return available[0], nil
}
func openDBusSession(ctx context.Context, conn *dbus.Conn) (dbus.ObjectPath, error) {
	var output dbus.Variant
	var path dbus.ObjectPath
	e := conn.Object(serviceName, servicePath).CallWithContext(ctx, serviceInterface+".OpenSession", 0, "plain", dbus.MakeVariant("")).Store(&output, &path)
	if e != nil || path == "/" || !path.IsValid() {
		return "", contracts.Fail("unavailable")
	}
	return path, nil
}
func closeDBusSession(ctx context.Context, conn *dbus.Conn, path dbus.ObjectPath) {
	conn.Object(serviceName, path).CallWithContext(ctx, "org.freedesktop.Secret.Session.Close", 0)
}
func (n linuxNative) get(ctx context.Context, id string) ([]byte, error) {
	conn, e := sessionBus(ctx)
	if e != nil {
		return nil, e
	}
	defer conn.Close()
	path, e := n.find(ctx, conn, id)
	if e != nil {
		return nil, e
	}
	session, e := openDBusSession(ctx, conn)
	if e != nil {
		return nil, e
	}
	defer closeDBusSession(ctx, conn, session)
	var secret dbusSecret
	e = conn.Object(serviceName, path).CallWithContext(ctx, itemInterface+".GetSecret", 0, session).Store(&secret)
	if e != nil || secret.Session != session {
		return nil, contracts.Fail("unavailable")
	}
	return secret.Value, nil
}
func rejectPrompt(ctx context.Context, conn *dbus.Conn, prompt dbus.ObjectPath) error {
	if prompt != "/" {
		if prompt.IsValid() {
			conn.Object(serviceName, prompt).CallWithContext(ctx, "org.freedesktop.Secret.Prompt.Dismiss", 0)
		}
		return contracts.Fail("unavailable")
	}
	return nil
}
func (n linuxNative) put(ctx context.Context, id string, value []byte) error {
	conn, e := sessionBus(ctx)
	if e != nil {
		return e
	}
	defer conn.Close()
	if e = unlocked(ctx, conn, collectionPath, collectionInterface); e != nil {
		return e
	}
	session, e := openDBusSession(ctx, conn)
	if e != nil {
		return e
	}
	defer closeDBusSession(ctx, conn, session)
	props := map[string]dbus.Variant{itemInterface + ".Label": dbus.MakeVariant("insonic credential " + id), itemInterface + ".Attributes": dbus.MakeVariant(map[string]string{"service": n.service, "username": id})}
	var item, prompt dbus.ObjectPath
	e = conn.Object(serviceName, collectionPath).CallWithContext(ctx, collectionInterface+".CreateItem", 0, props, dbusSecret{session, []byte{}, value, "application/octet-stream"}, true).Store(&item, &prompt)
	if e != nil {
		return contracts.Fail("unavailable")
	}
	if e = rejectPrompt(ctx, conn, prompt); e != nil {
		return e
	}
	if item == "/" || !item.IsValid() {
		return contracts.Fail("unavailable")
	}
	return nil
}
func (n linuxNative) remove(ctx context.Context, id string) error {
	conn, e := sessionBus(ctx)
	if e != nil {
		return e
	}
	defer conn.Close()
	path, e := n.find(ctx, conn, id)
	if e != nil {
		return e
	}
	var prompt dbus.ObjectPath
	e = conn.Object(serviceName, path).CallWithContext(ctx, itemInterface+".Delete", 0).Store(&prompt)
	if e != nil {
		return contracts.Fail("unavailable")
	}
	return rejectPrompt(ctx, conn, prompt)
}
