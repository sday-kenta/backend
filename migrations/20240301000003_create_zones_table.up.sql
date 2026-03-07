CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE IF NOT EXISTS zones (
    id serial PRIMARY KEY,
    name text NOT NULL UNIQUE,
    geom geometry(MultiPolygon, 4326) NOT NULL
);

CREATE INDEX IF NOT EXISTS zones_geom_idx ON zones USING GIST (geom);