CREATE TABLE post (
  id bigint NOT NULL,
  is_published boolean DEFAULT false,
  author bigint NOT NULL,
  location point NOT NULL,
  PRIMARY KEY(id));
CREATE INDEX by_location ON post USING gist(location);

CREATE TABLE message (
  id bigint NOT NULL,
  post bigint NOT NULL REFERENCES post(id) ON DELETE CASCADE,
  kind smallint NOT NULL,
  size bigint NOT NULL,
  is_published boolean DEFAULT false,
  data bytea,
  PRIMARY KEY(id, post));