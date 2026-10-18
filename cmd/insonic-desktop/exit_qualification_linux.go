//go:build desktop && linux

// SPDX-License-Identifier: Apache-2.0
package main

/*
#cgo pkg-config: gtk+-3.0
#include <gtk/gtk.h>

typedef struct { gint state; gint close_window; gint64 deadline; } InsonicExitProbe;
static GtkWidget *insonic_find_menu(GtkWidget *widget) {
 if (GTK_IS_MENU_BAR(widget)) return widget;
 if (!GTK_IS_CONTAINER(widget)) return NULL;
 GList *children=gtk_container_get_children(GTK_CONTAINER(widget));
 GtkWidget *found=NULL;
 for (GList *node=children;node && !found;node=node->next) found=insonic_find_menu(GTK_WIDGET(node->data));
 g_list_free(children);return found;
}
static GtkWidget *insonic_named_item(GtkWidget *shell,const char *label) {
 if (!GTK_IS_CONTAINER(shell)) return NULL;
 GList *children=gtk_container_get_children(GTK_CONTAINER(shell));GtkWidget *found=NULL;
 for (GList *node=children;node;node=node->next) {
  GtkWidget *item=GTK_WIDGET(node->data);
  if (GTK_IS_MENU_ITEM(item) && g_strcmp0(gtk_menu_item_get_label(GTK_MENU_ITEM(item)),label)==0) {
   if (found) {found=NULL;break;}found=item;
  }
 }
 g_list_free(children);return found;
}
static gboolean insonic_exit_tick(gpointer data) {
 InsonicExitProbe *probe=data;
 GList *windows=gtk_window_list_toplevels();GtkWidget *window=NULL;int count=0;
 for (GList *node=windows;node;node=node->next) {
  if (g_strcmp0(gtk_window_get_title(GTK_WINDOW(node->data)),"insonic")==0) {window=node->data;count++;}
 }
 g_list_free(windows);
 if (count==1) {
  GtkWidget *bar=insonic_find_menu(window);
  GtkWidget *file=bar?insonic_named_item(bar,"File"):NULL;
  GtkWidget *submenu=file?gtk_menu_item_get_submenu(GTK_MENU_ITEM(file)):NULL;
  GtkWidget *exit=submenu?insonic_named_item(submenu,"Exit"):NULL;
  if (exit && gtk_widget_get_sensitive(exit)) {
   if (probe->close_window) gtk_window_close(GTK_WINDOW(window));
   else gtk_menu_item_activate(GTK_MENU_ITEM(exit));
   g_atomic_int_set(&probe->state,1);return G_SOURCE_REMOVE;
  }
 }
 if (g_get_monotonic_time()>probe->deadline || count>1) {g_atomic_int_set(&probe->state,-1);return G_SOURCE_REMOVE;}
 return G_SOURCE_CONTINUE;
}
static InsonicExitProbe *insonic_exit_start(int close_window) {
 InsonicExitProbe *probe=g_new0(InsonicExitProbe,1);probe->close_window=close_window;
 probe->deadline=g_get_monotonic_time()+5000000;
 g_timeout_add(20,insonic_exit_tick,probe);return probe;
}
static int insonic_exit_state(InsonicExitProbe *probe) {return g_atomic_int_get(&probe->state);}
static void insonic_exit_free(InsonicExitProbe *probe) {g_free(probe);}
*/
import "C"

import (
	"fmt"
	"time"
)

func activateNativeExit(action string) error {
	if action != "menu" && action != "close" {
		return fmt.Errorf("unknown native Exit action")
	}
	closeWindow := C.int(0)
	if action == "close" {
		closeWindow = 1
	}
	probe := C.insonic_exit_start(closeWindow)
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		state := C.insonic_exit_state(probe)
		if state != 0 {
			C.insonic_exit_free(probe)
			if state == 1 {
				return nil
			}
			return fmt.Errorf("owned GTK File/Exit menu unavailable")
		}
		time.Sleep(20 * time.Millisecond)
	}
	// A stalled GTK loop may still own the scheduled callback. Retain its memory
	// until this bounded qualification process exits rather than freeing live data.
	return fmt.Errorf("native GTK action timed out")
}
