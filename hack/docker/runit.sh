#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

export PGHOST=127.0.0.1
export PGPORT=5432
export PGUSER=gearman
export PGPASSWORD=$(date +%s | sha256sum | base64 | head -c 32 ; echo)
export PGDATABASE=jobs

mkdir -p $PGROOT/schema

cat >$PGROOT/schema/createdb.sql <<EOL
CREATE ROLE $PGUSER  WITH LOGIN CREATEROLE ENCRYPTED PASSWORD '$PGPASSWORD';
CREATE DATABASE $PGDATABASE WITH ENCODING 'utf8' LC_COLLATE='en_US.UTF-8' LC_CTYPE='en_US.UTF-8' TEMPLATE=template0 OWNER $PGUSER;
EOL

cat >$PGROOT/schema/gearman-tables.sql <<EOL
-- Ref: http://gearman.info/gearmand/queues/postgres.html
START TRANSACTION;
SET standard_conforming_strings=off;
SET escape_string_warning=off;
SET CONSTRAINTS ALL DEFERRED;

-- DROP TABLE gearman_queue;
CREATE TABLE gearman_queue
  (
    --Since GEARMAN_UNIQUE_SIZE = 64 in libgearman/constants.h
    unique_key VARCHAR(64),
    function_name VARCHAR(255),
    priority INTEGER,
    data BYTEA,
    when_to_run INTEGER,
    PRIMARY KEY (unique_key),
    UNIQUE (unique_key, function_name)
);

-- Tables to store gearman cron job data --
-- DROP TABLE gearman_cron;
CREATE TABLE gearman_cron
  (
    "id" bigserial,
    "name" text NOT NULL,
    "jobId" BIGINT NOT NULL,
    "expression" text NOT NULL,
    "worker" text NOT NULL,
    "data" bytea,
    "successfulRun" BIGINT,
    "failedRun" BIGINT,
    "lastRun" BIGINT,
    "nextRun" BIGINT,
    "dateCreated" BIGINT NOT NULL,
    "isDeleted" SMALLINT NOT NULL,
    PRIMARY KEY ("id")
);

ALTER TABLE "gearman_queue" OWNER TO $PGUSER;
ALTER TABLE "gearman_cron" OWNER TO $PGUSER;

COMMIT;
EOL

cat >$PGROOT/schema/.setup-db.sh <<EOL
#!/bin/bash
set -x

psql postgres -f \$PGROOT/schema/createdb.sql
psql $PGDATABASE -f \$PGROOT/schema/gearman-tables.sql
EOL
chmod 755 $PGROOT/schema/.setup-db.sh
# This line will trigger postgres. So, always keep this as the last line for db setup operations.
mv $PGROOT/schema/.setup-db.sh $PGROOT/schema/setup-db.sh

# Wait for postgres to start
# ref: http://unix.stackexchange.com/a/5279
while ! nc -q 1 $PGHOST $PGPORT </dev/null; do sleep 5; done

export > /etc/envvars

echo "Starting runit..."
exec /usr/sbin/runsvdir-start
