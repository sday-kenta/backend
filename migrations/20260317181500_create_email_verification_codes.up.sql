CREATE TABLE IF NOT EXISTS email_verification_codes (
    id          BIGSERIAL PRIMARY KEY,
    email       VARCHAR(255) NOT NULL,
    purpose     VARCHAR(32)  NOT NULL,
    code        VARCHAR(16)  NOT NULL,
    expires_at  TIMESTAMPTZ  NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Only one active (not consumed) code per email+purpose
CREATE UNIQUE INDEX IF NOT EXISTS ux_email_verification_codes_active
    ON email_verification_codes (email, purpose)
    WHERE consumed_at IS NULL;

