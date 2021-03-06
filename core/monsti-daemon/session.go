// This file is part of Monsti, a web content management system.
// Copyright 2012-2014 Christian Neumann
//
// Monsti is free software: you can redistribute it and/or modify it under the
// terms of the GNU Affero General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option) any
// later version.
//
// Monsti is distributed in the hope that it will be useful, but WITHOUT ANY
// WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR
// A PARTICULAR PURPOSE.  See the GNU Affero General Public License for more
// details.
//
// You should have received a copy of the GNU Affero General Public License
// along with Monsti.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"code.google.com/p/go.crypto/bcrypt"
	"github.com/chrneumann/htmlwidgets"
	"github.com/gorilla/sessions"
	"pkg.monsti.org/gettext"
	"pkg.monsti.org/monsti/api/service"
	"pkg.monsti.org/monsti/api/util"
	"pkg.monsti.org/monsti/api/util/template"
)

type loginFormData struct {
	Login, Password string
}

// Login handles login requests.
func (h *nodeHandler) Login(c *reqContext) error {
	G, _, _, _ := gettext.DefaultLocales.Use("", c.UserSession.Locale)
	data := loginFormData{}

	form := htmlwidgets.NewForm(&data)
	form.AddWidget(new(htmlwidgets.TextWidget), "Login", G("Login"), "")
	form.AddWidget(new(htmlwidgets.PasswordWidget), "Password", G("Password"), "")

	switch c.Req.Method {
	case "GET":
	case "POST":
		c.Req.ParseForm()
		if form.Fill(c.Req.Form) {
			user, err := getUser(data.Login,
				h.Settings.Monsti.GetSiteDataPath(c.Site.Name))
			if err != nil {
				return fmt.Errorf("Could not get user: %v", err)
			}
			if user != nil && passwordEqual(user.Password, data.Password) {
				c.Session.Values["login"] = user.Login
				c.Session.Save(c.Req, c.Res)
				http.Redirect(c.Res, c.Req, c.Node.Path, http.StatusSeeOther)
				return nil
			}
			form.AddError("", G("Wrong login or password."))
		}
	default:
		return fmt.Errorf("Request method not supported: %v", c.Req.Method)
	}
	data.Password = ""
	body, err := h.Renderer.Render("actions/loginform", template.Context{
		"Form": form.RenderData()}, c.UserSession.Locale,
		h.Settings.Monsti.GetSiteTemplatesPath(c.Site.Name))
	if err != nil {
		return fmt.Errorf("Can't render login form: %v", err)
	}
	env := masterTmplEnv{Node: c.Node, Session: c.UserSession, Title: G("Login"),
		Description: G("Login with your site account."),
		Flags:       EDIT_VIEW}
	fmt.Fprint(c.Res, renderInMaster(h.Renderer, []byte(body), env, h.Settings,
		*c.Site, c.UserSession.Locale, c.Serv))
	return nil
}

// Logout handles logout requests.
func (h *nodeHandler) Logout(c *reqContext) error {
	delete(c.Session.Values, "login")
	c.Session.Save(c.Req, c.Res)
	http.Redirect(c.Res, c.Req, c.Node.Path, http.StatusSeeOther)
	return nil
}

// getSession returns a currently active or new session.
func getSession(r *http.Request, site util.SiteSettings) (
	*sessions.Session, error) {
	if len(site.SessionAuthKey) == 0 {
		return nil, fmt.Errorf(`Missing "SessionAuthKey" setting.`)
	}
	store := sessions.NewCookieStore([]byte(site.SessionAuthKey))
	session, _ := store.Get(r, "monsti-session")
	return session, nil
}

// getClientSession returns the client session for the given session.
//
// configDir is the site's configuration directory.
func getClientSession(session *sessions.Session,
	dataDir string) (uSession *service.UserSession, err error) {
	uSession = new(service.UserSession)
	loginData, ok := session.Values["login"]
	if !ok {
		return
	}
	login_, ok := loginData.(string)
	if !ok {
		delete(session.Values, "login")
		return
	}
	user, err := getUser(login_, dataDir)
	if err != nil {
		err = fmt.Errorf("Could not get user: %v", err)
		return
	}
	if user == nil {
		delete(session.Values, "login")
		return
	}
	*uSession = service.UserSession{User: user}
	return
}

// getUser returns the user with the given login.
func getUser(login_, dataDir string) (*service.User, error) {
	path := filepath.Join(dataDir, "users.json")
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Could not load user database: %v", err)
	}
	var users map[string]service.User
	if err = json.Unmarshal(content, &users); err != nil {
		return nil, fmt.Errorf("Could not unmarshal user database: %v", err)
	}
	if user, ok := users[login_]; ok {
		user.Login = login_
		return &user, nil
	}
	return nil, nil
}

// checkPermission checks if the session's user might perform the given action.
func checkPermission(action service.Action, session *service.UserSession) bool {
	auth := session.User != nil
	switch action {
	case service.RemoveAction, service.EditAction, service.AddAction,
		service.LogoutAction:
		if auth {
			return true
		}
	default:
		return true
	}
	return false
}

// passwordEqual returns true iff the hash matches the password.
func passwordEqual(hash, password string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(hash),
		[]byte(password)); err != nil {
		return false
	}
	return true
}
