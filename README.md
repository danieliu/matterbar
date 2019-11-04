# Matterbar

A Mattermost plugin integration with Rollbar webhook events.

## Installation

Follow the [Mattermost docs](https://docs.mattermost.com/administration/plugins.html#set-up-guide) if necessary.

1. Go to [releases](https://github.com/danieliu/matterbar/releases) and download the latest release.
2. Upload to your Mattermost server via **System Console -> Plugins -> Management**.

### Upgrading from < v1.0.0

Requirements:
* Mattermost Server `>= v5.12.0`
* Convert previously configured `username` into a `bot`. This can be done by running the CLI `mattermost user convert <username> --bot`. More info on bots are available [here](https://mattermost.com/pl/default-bot-accounts).

## Configuration / Usage

Configure the plugin itself in **System Console -> Plugins -> Matterbar**.

* `Default Team`: if this isn't specified, a `team` query parameter will be expected.
* `Default Channel`: if this isn't specified, a `channel` query parameter will be expected.
* `Auth Secret`: the auth string, expected in the `auth` query parameter to authenticate incoming requests.

* `Username`: (deprecated, only for `< v1.0.0`) the user that the plugin will post as upon receiving Rollbar webhook events.

At the very least, hit the auth secret **Regenerate** button. For versions `< v1.0.0`), input a **Username** as well. If any of these are misconfigured, e.g. a default team or channel does not exist, the plugin will log an error to the Mattermost system logs.

On the Rollbar side, configure your webhooks at

`https://rollbar.com/<user-or-organization>/<project>/settings/notifications/webhook/` .

Set the URL to point to your Mattermost instance, and the plugin's custom webhook endpoint at

`https://<mattermost-server-instance>.com/plugins/matterbar/notify?auth=<auth-secret>&team=<team-name>&channel=<channel-name>` .

For example:

https://mattermost.example.com/plugins/matterbar/notify?auth=YT5QclfXXrLyMDl-zw2bLv0aD0TlSX13&team=developers&channel=rollbars

`team` and `channel` can be omitted if the defaults have been configured. They can also be configured on a per webhook rule basis to customize where Rollbar notifications will be posted.

### /rollbar command

The plugin also comes with a custom slash command to set users who wish to @'d.

* `/rollbar list`: lists the users who will be @'d whenever an event is posted
* `/rollbar notify @username`: adds the user to the @ list
* `/rollbar remove @username`: removed the user from the @ list

The list of @'d users is separate per channel.

## Development

Follow the Mattermost [server developer setup](https://developers.mattermost.com/contribute/server/developer-setup/) and [webapp developer setup](https://developers.mattermost.com/contribute/webapp/developer-setup/). Install the plugin with `make deploy` assuming each repo is in the same dir.

E.g.
```bash
cd /path/to/workspace
git clone https://github.com/mattermost/mattermost-server.git
git clone https://github.com/mattermost/mattermost-webapp.git
git clone https://github.com/danieliu/matterbar.git

cd matterbar && make deploy
cd mattermost-server && make run
```

Navigate to [plugin management](http://localhost:8065/admin_console/plugins/plugin_matterbar) and configure matterbar.

### Testing

Run the tests with:

```bash
make test
```

Build the audit script and run it against localhost or an actual server:

```bash
make build-audit
./scripts/audit -a $AUTH_TOKEN  # audit localhost
./scripts/audit -a $AUTH_TOKEN -s https -h mattermost.server.com  # audit a server
```
