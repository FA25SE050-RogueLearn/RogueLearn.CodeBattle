-- This file store ENUM type
CREATE TYPE event_type AS ENUM (
    'event_unspecified',
    'code_battle',
    'workshop',
    'seminar',
    'social'
);

CREATE TYPE submission_status AS ENUM (
  'event_unspecified',
  'pending',
  'accepted',
  'wrong_answer',
  'limit_exceed',
  'runtime_error',
  'compilation_error'
);
