/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package plugin

import "context"

type Connector interface {
	Base

	// ConnectorLogoSVG presents the logo in svg format
	ConnectorLogoSVG() string

	// ConnectorName presents the name of the connector
	// e.g. Facebook, Twitter, Instagram
	ConnectorName() Translator

	// ConnectorSlugName presents the slug name of the connector
	// Please use lowercase and hyphen as the separator
	// e.g. facebook, twitter, instagram
	ConnectorSlugName() string

	// ConnectorSender presents the sender of the connector
	// It handles the start endpoint of the connector
	// receiverURL is the whole URL of the receiver
	ConnectorSender(ctx *GinContext, receiverURL string) (redirectURL string)

	// ConnectorReceiver presents the receiver of the connector
	// It handles the callback endpoint of the connector, and returns the
	ConnectorReceiver(ctx *GinContext, receiverURL string) (userInfo ExternalLoginUserInfo, err error)
}

// ExternalLoginUserInfo external login user info
type ExternalLoginUserInfo struct {
	// required. The unique user ID provided by the third-party login
	ExternalID string
	// optional. This name is used preferentially during registration
	DisplayName string
	// optional. This username is used preferentially during registration
	Username string
	// optional. If email exist will bind the existing user
	// IMPORTANT: The email must have been verified. If the plugin can't guarantee the email is verified, please leave it empty.
	Email string
	// optional. The avatar URL provided by the third-party login platform
	Avatar string
	// optional. The original user information provided by the third-party login platform
	MetaInfo string
	// optional. When true, Username is the upstream-validated, globally
	// unique handle and must be applied verbatim — no transform, no dedup
	// suffix, no random fallback. If it cannot be applied (invalid,
	// reserved, or collides) the login fails. Connectors set this when
	// the IdP owns the username namespace.
	UsernameAuthoritative bool
}

// ConnectorAfterLogin is an optional interface a Connector can implement to be
// notified after a successful login binds the external identity to a local
// user. Useful for IDPs that maintain a directory of (sub, app_user_id) pairs
// (e.g. fastgate) and need the local ID reported back. The hook is called once
// per login; errors are logged and do not block the user from signing in.
type ConnectorAfterLogin interface {
	AfterLogin(ctx context.Context, externalID, localUserID string) error
}

var (
	// CallConnector is a function that calls all registered connectors
	CallConnector,
	registerConnector = MakePlugin[Connector](false)
)
