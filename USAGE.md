# GitHub Archives to Postgres, InfluxDB, Grafana

Author: Łukasz Gryglicki <lukaszgryglick@o2.pl>

# Implemented in two languages (history)

This toolset was first implemented in Ruby with Postgres database.

Then MySQL support was added.

MySQL proved to be slower and harder to use than Postgres.

Entire toolset was rewritten in Go.

Go version only support Postgres, it proved to be a lot faster than Ruby version.

Finally, Ruby version was dropped.

This tools filter GitHub archive for given date period and given organization, repository and save results in a Postgres database.
It can also save results into JSON files.
It displays results using Grafana and InfluxDB time series database.

It can import developers affiliations from [cncf/gitdm](https://github.com/cncf/gitdm).

# Compilation

Uses GNU `Makefile`:
- `make check` - to apply gofmt, goimports, golint, go vet.
- `make` to compile static binaries: `structure`, `gha2db`, `db2influx`, `gha2db_sync`, `runq`, `z2influx`, `import_affs`, `annotations`.
- `make install` - to install binaries, this is needed for cron job.
- `make clean` - to clean binaries
- `make test` - to execute non-DB tests
- `PG_DB=dbtest PG_PASS=pwd IDB_DB=dbtest IDB_PASS=pwd make dbtest` - to execute DB tests.

All `*.go` files in project root directory are common library `gha2db` for all go executables.
All `*_test.go` and `test/*.go` are Go test files, that are used only for testing.

To run tools locally (without install) prefix them with `GHA2DB_LOCAl=1 `.

# Usage:

Local:
- `make`
- `ENV_VARIABLES GHA2DB_LOCAl=1 ./gha2db YYYY-MM-DD HH YYYY-MM-DD HH [org [repo]]`.

Installed:
- `make`
- `sudo make install`
- `ENV_VARIABLES gha2db YYYY-MM-DD HH YYYY-MM-DD HH [org [repo]]`.


You can use already populated Postgres dump: [Kubernetes Psql dump](https://cncftest.io/web/k8s.sql.xz) (more than 380 Mb, more than 7,5Gb uncompressed)

There is also a dump for `cncf` org: [CNCF Psql dump](https://cncftest.io/web/cncf.sql.xz) (less than 900 kb, about 8,5 Mb uncompressed, data from 2017-03-01)

First two parameters are date from:
- YYYY-MM-DD
- HH

Next two parameters are date to:
- YYYY-MM-DD
- HH

Both next two parameters are optional:
- org (if given and non-empty '' then only return JSONs matching given org). You can also provide a comma-separated list of orgs here: 'org1,org2,org3'.
- repo (if given and non-empty '' then only return JSONs matching given repo). You can also provide a comma-separated list of repos here: 'repo1,repo2'.

Org/Repo filtering:
- You can filter only by org by passing for example 'kubernetes' for org and '' for repo or skipping repo.
- You can filter only by repo, You need to pass '' as org and then repo name.
- You can return all JSONs by skipping both params.
- You can provide both to observe only events from given org/repo.
- You can list exact full repository names to run on: use `GHA2DB_EXACT=1` to process only repositories listed as "orgs" parameter, by their full names, like for example 3 repos: "GoogleCloudPlatform/kubernetes,kubernetes,kubernetes/kubernetes".

# Configuration

You can tweak `gha2db` tools by environment variables:
- Set `GHA2DB_ST` environment variable to run single threaded version.
- Set `GHA2DB_JSON` to save single events JSONs in [jsons/](https://github.com/cncf/gha2db/blob/master/jsons/) directory.
- Set `GHA2DB_NODB` to skip DB processing at all (if `GHA2DB_JSON` not set it will parse all data from GHA, but do nothing with it).
- Set `GHA2DB_DEBUG` set to 1 to see output for all events generated, set to 2 to see all SQL query parameters.
- Set `GHA2DB_QOUT` to see all SQL queries.
- Set `GHA2DB_MGETC` to "y" to assume "y" for `getchar` function (for example to answer "y" to `structure`'s Continue? question).
- Set `GHA2DB_CTXOUT` to display full environment context.
- Set `GHA2DB_NCPUS` to positive numeric value, to override the number of CPUs to run, this overwrites `GHA2DB_ST`.
- Set `GHA2DB_STARTDT`, to use start date for processing events (when syncing data with an empty database), default `2015-08-06 22:00 UTC`, expects format "YYYY-MM-DD HH:MI:SS".
- Set `GHA2DB_LASTSERIES`, to specify which InfluxDB series use to determine newest data (it will be used to query the newest timestamp), default `'all_prs_merged_d'`.
- Set `GHA2DB_CMDDEBUG` set to 1 to see commands executed, set to 2 to see commands executed and their output, set to 3 to see full exec environment.
- Set `GHA2DB_EXPLAIN` for `runq` tool, it will prefix query select(s) with "explain " to display query plan instead of executing the real query. Because metric can have multiple selects, and only main select should be replaced with "explain select" - we're replacing only downcased "select" statement followed by newline ("select\n" --> "explain select\n")
- Set `GHA2DB_OLDFMT` for `gha2db` tool to make it use old pre-2015 GHA JSONs format (instead of a new one used by GitHub Archives from 2015-01-01).
- Set `GHA2DB_EXACT` for `gha2db` tool to make it process only repositories listed as "orgs" parameter, by their full names, like for example 3 repos: "GoogleCloudPlatform/kubernetes,kubernetes,kubernetes/kubernetes"
- Set `GHA2DB_SKIPLOG` for any tool to skip logging output to `gha_logs` table.
- Set `GHA2DB_LOCAL` for gha2db_sync tool to make it prefix call to other tools with "./" (so it will use other tools binaries from the current working directory instead of `/usr/bin/`). Local mode uses "./metrics/" to search for metrics files. Otherwise "/etc/gha2db/metrics/" is used.

All environment context details are defined in [context.go](https://github.com/cncf/gha2db/blob/master/context.go), please see that file for details (You can also see how it works in [context_test.go](https://github.com/cncf/gha2db/blob/master/context_test.go)).

Examples in this shell script (some commented out, some not):

`time PG_PASS=your_pass ./gha2db.sh`

# Informations

GitHub archives keep data as Gzipped JSONs for each hour (24 gzipped JSONs per day).
Single JSON is not a real JSON file, but "\n" newline separated list of JSONs for each GitHub event in that hour.
So this is a JSON array in reality.

GihHub archive files can be found here <https://www.githubarchive.org>

For example to fetch 2017-08-03 18:00 UTC can be fetched by:

- wget http://data.githubarchive.org/2017-08-03-18.json.gz

Gzipped files are usually 10-30 Mb in size (single hour).
Decompressed fields are usually 100-200 Mb.

We download this gzipped JSON, process it on the fly, creating the array of JSON events and then each single event JSON matching org/repo criteria is saved in [jsons](https://github.com/cncf/gha2db/blob/master/jsons/) directory as `N_ID.json` where:
- N - given GitHub archive''s JSON hour as UNIX timestamp.
- ID - GitHub event ID.

Once saved, You can review those JSONs manually (they're pretty printed).

# Multithreading

For example <http://cncftest.io> server has 48 CPU cores.
It will just process 48 hours in parallel.
It detects the number of available CPUs automatically.
You can use `GHA2DB_ST` environment variable to force single threaded version.

# Results (JSON)

Usually, there are about 25000 GitHub events in a single hour in Jan 2017 (for July 2017 it is 40000).
Average seems to be from 15000 to 60000.

1) Running this program on 5 days of data with org `kubernetes` (and no repo set - which means all kubernetes repos):
- Takes: 10 minutes 50 seconds.
- Generates 12002 JSONs with summary size 165 Mb (each JSON is a single GitHub event).
- To do so it processes about 21 Gb of data.

2) Running this program 1 month of data with org `kubernetes` (and no repo set - which means all kubernetes repos).
June 2017:
- Takes: 61 minutes 26 seconds.
- Generates 60773 JSONs with summary size 815 Mb.
- To do so it processes about 126 Gb of data.

3) Running this program 3 hours of data with no filters.
2017-07-05 hours: 18, 19, 20:
- Takes: 55 seconds.
- Generates 168683 JSONs with summary size 1.1 Gb.
- To do so it processes about 126 Gb of data.

Taking all events from a single day is 5 minutes 50 seconds (2017-07-28):
- Generates 1194599 JSON files (1.2M)
- Takes 7 Gb of disc space

There is a big file containing all Kubernetes events JSONs from Aug 2017 concatenated and xzipped: [K8s August 2017](https://cncftest.io/web/k8s_201708.json.xz).

Please note that this is not a correct JSON, it contains files separated by line `JSON: jsons/filename.json` - that says what was the original JSON filename. This file is 16.7M xzipped, but 1.07G uncompressed.

Please also note that JSON for 2016-10-21 18:00 is broken, so running this command will produce no data. The code will output error to logs and continue. Always examine `errors.txt` from `kubernetes*.sh` script.

This will log error and process no JSONs:
- `./gha2db 2016-10-21 18 2016-10-21 18 'kubernetes,kubernetes-client,kubernetes-incubator'`.

1) Running on all Kubernetes org since the beginning of kubernetes:
- Takes about 2h15m.
- Database dump is 7.5 Gb, XZ compressed dump is ~400 Mb
- Note that those counts include historical changes to objects (for example single issue can have multiple entries with a different state on different events)
- Completed dump <https://cncftest.io/web/k8s.sql.xz>

# PostgreSQL database setup

Detailed setup instructions are here (they use already populated postgres dump):
- [Mac](https://github.com/cncf/gha2db/blob/master/INSTALL_MAC.md)
- [Linux](https://github.com/cncf/gha2db/blob/master/INSTALL_UBUNTU.md)

In short for Ubuntu like Linux:

- apt-get install postgresql 
- sudo -i -u postgres
- psql
- create database gha;
- create user gha_admin with password 'your_password_here';
- grant all privileges on database "gha" to gha_admin;
- alter user gha_admin createdb;
- go get github.com/lib/pq
- PG_PASS='pwd' ./structure
- psql gha

`structure` script is used to create Postgres database schema.
It gets connection details from environmental variables and falls back to some defaults.

Defaults are:
- Database host: environment variable PG_HOST or `localhost`
- Database port: PG_PORT or 5432
- Database name: PG_DB or 'gha'
- Database user: PG_USER or 'gha_admin'
- Database password: PG_PASS || 'password'
- Database SSL: PG_SSL || 'disable'
- If You want it to generate database indexes set `GHA2DB_INDEX` environment variable
- If You want to skip table creations set `GHA2DB_SKIPTABLE` environment variable (when `GHA2DB_INDEX` also set, it will create indexes on already existing table structure, possibly already populated)
- If You want to skip creating DB tools (like views and functions), use `GHA2DB_SKIPTOOLS` environment variable.

It is recommended to create structure without indexes first (the default), then get data from GHA and populate array, and finally add indexes. To do do:
- `time PG_PASS=your_password ./structure`
- `time PG_PASS=your_password ./gha2db.sh`
- `time GHA2DB_SKIPTABLE=1 GHA2DB_INDEX=1 PG_PASS=your_password ./structure` (will take some time to generate indexes on populated database)

Typical internal usage:
`time GHA2DB_INDEX=1 PG_PASS=your_password ./structure`

Alternatively, you can use [structure.sql](https://github.com/cncf/gha2db/blob/master/structure.sql) to create database structure.

You can also use already populated Postgres dump: [Kubernetes Psql dump](https://cncftest.io/web/k8s.sql.xz)

# Database structure

You can see database structure in [structure.go](https://github.com/cncf/gha2db/blob/master/structure.go)/[structure.sql](https://github.com/cncf/gha2db/blob/master/structure.sql).

The main idea is that we divide tables into 2 groups:
- const: meaning that data in this table is not changing in time (is saved once)
- variable: meaning that data in those tables can change between GH events, and GH event_id is a part of this tables primary key.

List of tables:
- `gha_actors`: const, users table
- `gha_actors_emails`: const, holds one or more email addresses for actors, this is filled by `./import_affs` tool.
- `gha_actors_affiliations`: const, holds one or more company affiliations for actors, this is filled by `./import_affs` tool.
- `gha_assets`: variable, assets
- `gha_branches`: variable, branches data
- `gha_comments`: variable (issue, PR, review)
- `gha_commits`: variable, commits
- `gha_companies`: const, companies, this is filled by `./import_affs` tool.
- `gha_events`: const, single GitHub archive event
- `gha_forkees`: variable, forkee, repo state
- `gha_issues`: variable, issues
- `gha_issues_assignees`: variable, issue assignees
- `gha_issues_labels`: variable, issue labels
- `gha_labels`: const, labels
- `gha_milestones`: variable, milestones
- `gha_orgs`: const, orgs
- `gha_pages`: variable, pages
- `gha_payloads`: const, event payloads
- `gha_pull_requests`: variable, pull requests
- `gha_pull_requests_assignees`: variable pull request assignees
- `gha_pull_requests_requested_reviewers`: variable, pull request requested reviewers
- `gha_releases`: variable, releases
- `gha_releases_assets`: variable, release assets
- `gha_repos`: const, repos
- `gha_teams`: variable, teams
- `gha_teams_repositories`: variable, teams repositories connections
- `gha_logs`: this is a table that holds all tools logs (unless `GHA2DB_SKIPLOG` is set)
- `gha_texts`: this is a compute table, that contains texts from comments, commits, issues and pull requests, updated by gha2db_sync and structure tools
- `gha_issues_pull_requests`: this is a compute table that contains PRs and issues connections, updated by gha2db_sync and structure tools
- `gha_issues_events_labels`: this is a compute table, that contains shortcuts to issues labels (for metrics speedup), updated by gha2db_sync and structure tools

There is some data duplication in various columns. This is to speedup metrics processing.
Such columns are described as "dup columns" in [structure.go](https://github.com/cncf/gha2db/blob/master/structure.go)
Such columns are prefixed by "dup_". They're usually not null columns, but there can also be null able columns - they start with "dupn_".

There is a standard duplicate event structure consisting of (dup_type, dup_actor_id, dup_actor_login, dup_repo_id, dup_repo_name, dup_created_at), I'll call it `eventd`

Duplicated columns:
- dup_actor_login, dup_repo_name in `gha_events` are taken from `gha_actors` and `gha_repos` to save joins.
- `eventd` on `gha_payloads`
- Just take a look at "dup_" and "dupn_" fields on all tables.

# Adding columns to existing database

- alter table table_name add col_name col_def;
- update ...
- alter table table_name alter column col_name set not null;

# JSON examples

There are examples of all kinds of GHA events JSONs in [./analysis](https://github.com/cncf/gha2db/blob/master/analysis/) directory.
There is also a file [analysis/analysis.txt](https://github.com/cncf/gha2db/blob/master/analysis/analysis.txt) that describes JSON structure analysis.

It was used very intensively during a development of SQL table structure.

All JSON and TXT files starting with "old_" and txt files starting with "old_" are the result of pre-2015 GHA JSONs structure analysis.

All JSON and TXT files starting with "new_" and txt files starting with "new_" are the result of new 2015+ GHA JSONs structure analysis.

To Run JSON structure analysis for either pre or from 2015 please do:
- `analysis_from2015.sh`.
- `analysis_pre2015.sh`.

Both those tools require Ruby. This tool was originally in Ruby, and there is no sense to rewrite it in Go because:
- It uses a very dynamic code, reflection and code evaluation as provided by properties list from the command line.
- It is used only during implementation (first post 2015 version, and now for pre-2015).
- It would be at least 10x longer and more complicated in Go, and probably not really faster because it would have to use reflection too.
- This kind of code will be very hard to read in Go.

# Running on Kubernetes

Kubernetes consists of 3 different orgs (from 2015-08-06 22:00), so to gather data for Kubernetes You need to provide them comma separated.

Before 2015-08-06 Kubernetes is in `GoogleCloudPlatform/kubernetes` or just few kubernetes repos without org. To process them You need to use special list mode `GHA2DB_EXACT`.

And finally before 2015-01-01 GitHub used different JSONs format. To process them You have to use `GHA2DB_OLDFMT` mode.

For example June 2017:
- `time PG_PASS=pwd ./gha2db 2017-06-01 0 2017-07-01 0 'kubernetes,kubernetes-incubator,kubernetes-client'`

To process kubernetes all time just use `kubernetes.sh` script. Like this:
- `time PG_PASS=pwd ./kubernetes.sh`.

# Metrics tool
There is a tool `runq`. It is used to compute metrics saved in `*.sql` files.
Please be careful when creating metric files, that needs to support `explain` mode (please see `GHA2DB_EXPLAIN` environment variable description):

Because metric can have multiple selects, and only main select should be replaced with "explain select" - we're replacing only lower case "select" statement followed by new line.
Exact match "select\n". Please see [metrics/reviewers.sql](https://github.com/cncf/gha2db/blob/master/metrics/reviewers.sql) to see how it works.

Metrics are in [./metrics/](https://github.com/cncf/gha2db/blob/master/metrics/) directory.

This tool takes at least one parameter - sql file name.

Typical usages:
- `time PG_PASS='password' ./runq metrics/metric.sql`

Some SQLs files require parameter substitution (like all metrics used by Grafana).

They usually have `'{{from}}'` and `'{{to}}'` parameters, to run such files do:
- `time PG_PASS='password' ./runq metrics/metric.sql '{{from}}' 'YYYY-MM-DD HH:MM:SS' '{{to}}' 'YYYY-MM-DD HH:MM:SS'`

You can also change any other value, just note that parameters after SQL file name are pairs: (`value_to_replace`, `replacement`).

# Sync tool

When You have imported all data You need - it needs to be updated periodically.
GitHub archive generates new file every hour.

Use `gha2db_sync` and/or `sync.sh` tools to update all Your data.

Example call:
- `PG_PASS='pwd' IDB_PASS='pwd' ./sync.sh`
- Add `GHA2DB_RESETIDB` environment variable to rebuild InfluxDB stats instead of update since the last run
- Add `GHA2DB_SKIPIDB` environment variable to skip syncing InfluxDB (so it will only sync Postgres DB)

There is a manual script that can be used to loop sync every defined number of seconds, for example for sync every 30 minutes:
- `PG_PASS='pwd' IDB_PASS='pwd' ./syncer.sh 1800`


Sync tool uses [gaps.yaml](https://github.com/cncf/gha2db/blob/master/metrics/gaps.yaml), to prefill some series with zeros. This is needed for metrics (like SIG mentions or PRs merged) that return multiple rows, depending on data range.

# Cron

You can install a cron job to run `gha2db_sync` automatically, please check "cron" section:
- [Mac](https://github.com/cncf/gha2db/blob/master/INSTALL_MAC.md)
- [Linux](https://github.com/cncf/gha2db/blob/master/INSTALL_UBUNTU.md)

# Developers affiliations

You need to get [github_users.json](https://raw.githubusercontent.com/cncf/gitdm/master/github_users.json) file from [CNCF/gitdm](https://github.com/cncf/gitdm).

To generate this file follow instructions on cncf/gitdm, or just get the newest version.

This file contains all GitHub user name - company affiliations found by `cncf/gitdm`.

To load it into our database use:
- `PG_PASS=pwd ./import_affs github_users.json`

# Repository groups

There are some groups of repositories that can be used to create metrics for lists of repositories.
Repository group is defined on `gha_repos` table using `repo_group` value.

To setup default repository groups:
- `PG_PASS=pwd ./setup_repo_groups.sh`.

This is a part of `kubernetes.sh` script and [kubernetes psql dump](https://cncftest.io/web/k8s.sql.xz) already has groups configured.

# Grafana output

You can visualise data using Grafana, see [grafana/](https://github.com/cncf/gha2db/blob/master/grafana/) directory:

# Install Grafana using:

Grafana install instruction are here:
- [Mac](https://github.com/cncf/gha2db/blob/master/INSTALL_MAC.md)
- [Linux](https://github.com/cncf/gha2db/blob/master/INSTALL_UBUNTU.md)

# To drop & recreate InfluxDB:
- `IDB_PASS='idb_password' ./grafana/influxdb_recreate.sh`
- `GHA2DB_RESETIDB=1 PG_PASS='pwd' IDB_PASS='pwd' ./sync.sh`
- Then eventually start syncer: `PG_PASS='pwd' IDB_PASS='pwd' ./syncer.sh 1800`

Or automatically: drop & create Influx DB, update Postgres DB since the last run, full populate InfluxDB, start syncer every 30 minutes:
- `IDB_PASS=pwd PG_PASS=pwd ./reinit_all.sh`

# Alternate solution with Docker:

Note that this is an old solution that worked, but wasn't tested recently.

- Start Grafana using `GRAFANA_PASS='password' grafana/grafana_start.sh` to install Grafana & InfluxDB as docker containers (this requires Docker).
- Start InfluxDB using `IDB_PASS='password' ./grafana/influxdb_setup.sh`, this requires Docker & previous command succesfully executed.
- To cleanup Docker Grafana image and start from scratch use `./grafana/docker_cleanup.sh`. This will not delete Your grafana config because it is stored in local volume `/var/lib/grafana`.
- To recreate all Docker Grafana/InfluxDB stuff from scratch do: `GRAFANA_PASS='' IDB_PASS='' GHA2DB_RESETIDB=1 PG_PASS='' IDB_PASS='' ./grafana/reinit.sh`

# Manually feeding InfluxDB & Grafana:

Feed InfluxDB using:
- `PG_PASS='psql_pwd' IDB_PASS='influxdb_pwd' ./db2influx sig_metions_data metrics/sig_mentions.sql '2017-08-14' '2017-08-21' d`
- The first parameter is used as exact series name when metrics query returns single row with single column value.
- First parameter is used as function name when metrics query return mutiple rows, each with >= 2 columns. This function receives data row and the period name and should return series name and value(s).
- The second parameter is a metrics SQL file, it should contain time conditions defined as `'{{from}}'` and `'{{to}}'`.
- Next two parameters are date ranges.
- The last parameter can be h, d, w, m, q, y (hour, day, week, month, quarter, year).
- This tool uses environmental variables starting with `IDB_`, please see `context.go`, `idb_conn.go` and `cmd/db2influx/db2influx.go` for details.
- `IDB_` variables are exactly the same as `PG_` to set host, database, user name, password.
- There is also `z2influx` tool. It is used to fill given series with zeros. Typical usage: `./z2influx 'series1,series2' 2017-01-01 2018-01-01 w` - will fill all weeks from 2017 with zeros for series1 and series2.
- `annotations` tool adds variuos data annotations that can be used in Grafana charts. It uses [annotations.yaml](https://github.com/cncf/gha2db/blob/master/metrics/annotations.yaml) file to define them, syntax is self describing. 

# To check results in the InfluxDB:
- influx
- auth (gha_admin/influxdb_pwd)
- use gha
- select * from reviewers
- select count(*) from reviewers
- show tag keys
- show field keys
- show series

# To drop data from InfluxDB:
- drop measurement reviewers
- drop series from reviewers

# Grafana dashboards
Grafana allows saving dashboards to JSON files.
There are few defined dashboards in [grafana/dashboards/](https://github.com/cncf/gha2db/blob/master/grafana/dashboards/) directory.

Metrics are described in [README](https://github.com/cncf/gha2db/blob/master/README.md) in `Grafana dashboards` and `Adding new metrics` sections.

# To enable SSL Grafana:
- First You need to install certbot, this is for example for Apache on Ubuntu 17.04:
- `sudo apt-get update`
- `sudo apt-get install software-properties-common`
- `sudo add-apt-repository ppa:certbot/certbot`
- `sudo apt-get update`
- `sudo apt-get install python-certbot-apache`
- `sudo certbot --apache`
- Then You need to proxy Apache https/SSL on port 443 to http on port 3000 (this is where Grafana listens)
- Your Grafana lives in https://<your_domain> (and https is served by Apache proxy to Grafana https:443 -> http:3000)
- Modified Apache config files are in [grafana/apache](https://github.com/cncf/gha2db/blob/master/grafana/apache/), You need to check them and enable something similar on Your machine.
- Please note that those modified Apache files additionally allows to put You website in `/web` path (this path is in exception list and is not proxied to Grafana), so You can for instance put [database dump](https://cncftest.io/web/k8s.sql.xz) there.
- Files in `[grafana/apache](https://github.com/cncf/gha2db/blob/master/grafana/apache/) should be copied to `/etc/apache2` (see comments starting with `LG:`) and then `service apache2 restart`

# Grafana anonymous login

Please see instructions [here](https://github.com/cncf/gha2db/blob/master/GRAFANA.md)

# Benchmarks
Benchmarks were executed on historical Ruby version and current Go version.

Please see [Benchmarks](https://github.com/cncf/gha2db/blob/master/BENCHMARK.md)

# Testing

Please see [Tests](https://github.com/cncf/gha2db/blob/master/TESTING.md)

