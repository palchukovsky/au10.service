CREATE TABLE post (
	id bigint NOT NULL,
  author bigint NOT NULL,
  location point,
	PRIMARY KEY(id));
CREATE INDEX by_location ON post USING gist(location);

CREATE TABLE message (
	id bigint NOT NULL,
  post bigint NOT NULL REFERENCES post(id) ON DELETE CASCADE,
  kind smallint NOT NULL,
  content bytea,
	PRIMARY KEY(id));
CREATE INDEX by_kind ON message(kind);