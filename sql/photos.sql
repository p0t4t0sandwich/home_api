CREATE TABLE photos (
    id TEXT NOT NULL PRIMARY KEY,
    file TEXT NOT NULL,
    ext TEXT NOT NULL,
    hash TEXT NOT NULL,
    phash BYTEA NOT NULL,
    description TEXT,
    source TEXT,
    subjects TEXT[],
    tags TEXT[],
    resolution TEXT,
    taken_at TIMESTAMP WITH TIME ZONE,
    uploaded_at TIMESTAMP WITH TIME ZONE NOT NULL,
    modified_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- https://stackoverflow.com/questions/17739887/how-to-xor-md5-hash-values-and-cast-them-to-hex-in-postgresql
CREATE FUNCTION xor_digests(_in1 bytea, _in2 bytea) RETURNS bytea
AS $$
DECLARE
 o int; -- offset
BEGIN
  FOR o IN 0..octet_length(_in1)-1 LOOP
    _in1 := set_byte(_in1, o, get_byte(_in1, o) # get_byte(_in2, o));
  END LOOP;
 RETURN _in1;
END;
$$ language plpgsql;
