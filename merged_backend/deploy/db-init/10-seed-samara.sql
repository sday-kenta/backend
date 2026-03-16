INSERT INTO zones (name, display_name, geom)
SELECT
    'samara',
    'Самара',
    ST_SetSRID(
        ST_GeomFromGeoJSON(geom::text),
        4326
    )
FROM (
    SELECT json_array_elements(json->'features')->'geometry' AS geom
    FROM (
        SELECT pg_read_file('/seed-data/samara.geojson')::json AS json
    ) t
) f
WHERE geom->>'type' = 'MultiPolygon'
  AND NOT EXISTS (
      SELECT 1 FROM zones WHERE name = 'samara'
  );
