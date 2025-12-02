-- Миграция для чистой PostgreSQL-базы (без PostGIS)
-- Внимание: запускать как суперпользователь или роль с достаточными правами.
-- BEGIN/COMMIT для атомарности (можно убрать если планируете частями)
BEGIN;

-------------------------
-- 0. Препроцесс: настройки
-------------------------
SET client_min_messages = NOTICE;
SET standard_conforming_strings = on;

-------------------------
-- 1. Расширения
-------------------------
-- crypt() и генерация UUID
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "citext";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-------------------------
-- 2. Схемы (опционально)
-------------------------
-- Используем public; при желании можно создать схему app
-- CREATE SCHEMA IF NOT EXISTS app;

-------------------------
-- 3. Типы ENUM
-------------------------
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role_t') THEN
CREATE TYPE user_role_t AS ENUM ('volunteer', 'organizer', 'admin');
END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'event_status_t') THEN
CREATE TYPE event_status_t AS ENUM (
      'draft','pending_moderation','active','cancelled','finished','rejected'
    );
END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'registration_status_t') THEN
CREATE TYPE registration_status_t AS ENUM (
      'applied','confirmed','rejected','cancelled','attended','no_show'
    );
END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'event_importance_t') THEN
CREATE TYPE event_importance_t AS ENUM ('low','medium','high','critical');
END IF;
END$$;

-------------------------
-- 4. Таблицы: пользователи, сессии, роли
-------------------------
CREATE TABLE IF NOT EXISTS users (
                       id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                       email CITEXT UNIQUE NOT NULL,
                       password_hash TEXT NOT NULL,
                       full_name TEXT,
                       phone TEXT,
                       bio TEXT,
                       created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                       updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                       is_blocked BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS user_roles (
                            user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                            role user_role_t NOT NULL,
                            granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                            PRIMARY KEY (user_id, role)
);

CREATE TABLE IF NOT EXISTS user_sessions (
                               id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                               user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                               refresh_token TEXT UNIQUE NOT NULL,
                               user_agent TEXT,
                               ip_address INET,
                               created_at TIMESTAMPTZ DEFAULT now(),
                               expires_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS organizer_verifications (
                                         id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                         user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                         company_name TEXT,
                                         documents JSONB,
                                         status TEXT NOT NULL DEFAULT 'pending',
                                         reviewed_by UUID REFERENCES users(id),
                                         reviewed_at TIMESTAMPTZ,
                                         created_at TIMESTAMPTZ DEFAULT now()
);

-------------------------
-- 5. Категории, теги, модерация
-------------------------
CREATE TABLE IF NOT EXISTS categories (
                            id SERIAL PRIMARY KEY,
                            name TEXT NOT NULL UNIQUE,
                            slug TEXT NOT NULL UNIQUE,
                            description TEXT,
                            created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tags (
                      id SERIAL PRIMARY KEY,
                      name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS moderation_logs (
                                 id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                 object_type TEXT NOT NULL,  -- e.g. 'event','user'
                                 object_id UUID,
                                 action TEXT NOT NULL,
                                 details JSONB,
                                 performed_by UUID REFERENCES users(id),
                                 performed_at TIMESTAMPTZ DEFAULT now()
);

-------------------------
-- 6. События (events)
-- храним location как TEXT и координаты как numeric (опционально)
-------------------------
CREATE TABLE IF NOT EXISTS events (
                        id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                        organizer_id UUID REFERENCES users(id) ON DELETE SET NULL,
                        title TEXT NOT NULL,
                        short_description TEXT,
                        full_description TEXT,
                        category_id INT REFERENCES categories(id),
                        importance event_importance_t NOT NULL DEFAULT 'medium',
                        status event_status_t NOT NULL DEFAULT 'draft',
                        capacity INT,
                        participants_count INT NOT NULL DEFAULT 0,
                        starts_at TIMESTAMPTZ,
                        ends_at TIMESTAMPTZ,
                        created_at TIMESTAMPTZ DEFAULT now(),
                        updated_at TIMESTAMPTZ DEFAULT now(),
                        address TEXT,
    -- optional lat/lon for simple geo (no PostGIS)
                        latitude NUMERIC(10,6),
                        longitude NUMERIC(10,6),
                        location_text_search TSVECTOR,
                        extra JSONB DEFAULT '{}',
                        moderated_by UUID REFERENCES users(id),
                        moderated_at TIMESTAMPTZ,
                        deleted_at TIMESTAMPTZ
);

-------------------------
-- 7. Теги для мероприятий (many-to-many)
-------------------------
CREATE TABLE IF NOT EXISTS event_tags (
                            event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
                            tag_id INT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
                            PRIMARY KEY (event_id, tag_id)
);

-------------------------
-- 8. Регистрации / участники
-------------------------
CREATE TABLE IF NOT EXISTS registrations (
                               id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                               event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
                               user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                               status registration_status_t NOT NULL DEFAULT 'applied',
                               applied_at TIMESTAMPTZ DEFAULT now(),
                               updated_at TIMESTAMPTZ DEFAULT now(),
                               comment TEXT,
                               seat_number INT,
                               UNIQUE (event_id, user_id)
);

-------------------------
-- 9. Отзывы и рейтинги
-------------------------
CREATE TABLE IF NOT EXISTS reviews (
                         id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                         event_id UUID REFERENCES events(id) ON DELETE CASCADE,
                         author_id UUID REFERENCES users(id) ON DELETE SET NULL,
                         rating SMALLINT NOT NULL CHECK (rating >= 1 AND rating <= 5),
                         comment TEXT,
                         created_at TIMESTAMPTZ DEFAULT now(),
                         is_visible BOOLEAN DEFAULT TRUE,
                         moderated_by UUID REFERENCES users(id),
                         moderated_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS user_ratings (
                              user_id UUID PRIMARY KEY REFERENCES users(id),
                              rating_score NUMERIC(8,4) NOT NULL DEFAULT 0,
                              rating_count INT NOT NULL DEFAULT 0,
                              last_updated TIMESTAMPTZ DEFAULT now()
);

-------------------------
-- 10. Уведомления
-------------------------
CREATE TABLE IF NOT EXISTS notifications (
                               id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                               user_id UUID REFERENCES users(id) ON DELETE CASCADE,
                               event_id UUID REFERENCES events(id),
                               type TEXT NOT NULL,
                               payload JSONB,
                               is_read BOOLEAN DEFAULT FALSE,
                               created_at TIMESTAMPTZ DEFAULT now()
);

-------------------------
-- 11. Materialized view: системная статистика
-------------------------
DROP MATERIALIZED VIEW IF EXISTS mv_system_stats;

CREATE MATERIALIZED VIEW mv_system_stats AS
SELECT
    (SELECT COUNT(*) FROM users) AS total_users,
    (SELECT COUNT(*) FROM events) AS total_events,
    (SELECT COUNT(DISTINCT user_id) FROM registrations WHERE status IN ('confirmed','attended')) AS active_volunteers,
    now() AS generated_at;

-------------------------
-- 12. Функции общего назначения
-------------------------
-- Функция обновления поля updated_at (используется ниже)
CREATE OR REPLACE FUNCTION trigger_update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Функция пересчёта user rating (на базе видимых отзывов)
CREATE OR REPLACE FUNCTION recalc_user_rating(p_user_id UUID)
RETURNS VOID AS $$
DECLARE
avg_rating NUMERIC;
  cnt INT;
BEGIN
SELECT COALESCE(AVG(r.rating),0), COALESCE(COUNT(r.*),0)
INTO avg_rating, cnt
FROM reviews r
         JOIN registrations reg ON reg.event_id = r.event_id AND reg.user_id = r.author_id
WHERE reg.user_id = p_user_id AND r.is_visible;

INSERT INTO user_ratings (user_id, rating_score, rating_count, last_updated)
VALUES (p_user_id, avg_rating, cnt, now())
    ON CONFLICT (user_id) DO UPDATE
                                 SET rating_score = EXCLUDED.rating_score,
                                 rating_count = EXCLUDED.rating_count,
                                 last_updated = now();
END;
$$ LANGUAGE plpgsql;

-- Функция для отправки уведомления (создает запись и вызывает pg_notify)
CREATE OR REPLACE FUNCTION send_notification(p_user_id UUID, p_event_id UUID, p_type TEXT, p_payload JSONB)
RETURNS VOID AS $$
DECLARE
nid UUID;
BEGIN
  nid := uuid_generate_v4();
INSERT INTO notifications(id, user_id, event_id, type, payload, created_at)
VALUES (nid, p_user_id, p_event_id, p_type, p_payload, now());

PERFORM pg_notify('notifications_channel',
    json_build_object('notification_id', nid, 'user_id', p_user_id, 'type', p_type, 'payload', p_payload)::text
  );
END;
$$ LANGUAGE plpgsql;

-------------------------
-- 13. Функции: регистрация, аутентификация, управление заявками
-------------------------
-- Создание пользователя (с хэшем пароля bcrypt)
CREATE OR REPLACE FUNCTION create_user(p_email TEXT, p_password TEXT, p_full_name TEXT)
RETURNS UUID AS $$
DECLARE
uid UUID;
  hashed TEXT;
BEGIN
  hashed := crypt(p_password, gen_salt('bf'));
INSERT INTO users(email, password_hash, full_name, created_at, updated_at)
VALUES (lower(p_email), hashed, p_full_name, now(), now())
    RETURNING id INTO uid;
RETURN uid;
EXCEPTION WHEN unique_violation THEN
  RAISE EXCEPTION 'Email already registered';
END;
$$ LANGUAGE plpgsql;

-- Авторизация + сессия
CREATE OR REPLACE FUNCTION authenticate_user(p_email TEXT, p_password TEXT, p_user_agent TEXT, p_ip INET, p_session_ttl INTERVAL)
RETURNS TABLE (user_id UUID, session_id UUID, expires_at TIMESTAMPTZ) AS $$
DECLARE
uid UUID;
  pass_hash TEXT;
  sid UUID;
  expires TIMESTAMPTZ;
BEGIN
SELECT id, password_hash INTO uid, pass_hash FROM users WHERE email = lower(p_email) AND NOT is_blocked;
IF NOT FOUND THEN
    RETURN;
END IF;

  IF pass_hash IS NULL OR crypt(p_password, pass_hash) <> pass_hash THEN
    RETURN;
END IF;

  sid := uuid_generate_v4();
  expires := now() + COALESCE(p_session_ttl, '30 days'::interval);

INSERT INTO user_sessions(id, user_id, refresh_token, user_agent, ip_address, created_at, expires_at)
VALUES (sid, uid, uuid_generate_v4()::text, p_user_agent, p_ip, now(), expires);

RETURN QUERY SELECT uid, sid, expires;
END;
$$ LANGUAGE plpgsql;

-- Подать заявку на событие
CREATE OR REPLACE FUNCTION apply_to_event(p_event_id UUID, p_user_id UUID, p_comment TEXT)
RETURNS VOID AS $$
BEGIN
  PERFORM 1 FROM events WHERE id = p_event_id AND status = 'active';
  IF NOT FOUND THEN
    RAISE EXCEPTION 'Event is not active or does not exist';
END IF;

INSERT INTO registrations(event_id, user_id, status, applied_at, comment)
VALUES (p_event_id, p_user_id, 'applied', now(), p_comment)
    ON CONFLICT (event_id, user_id) DO UPDATE
                                           SET status = 'applied', updated_at = now(), comment = EXCLUDED.comment;

PERFORM send_notification((SELECT organizer_id FROM events WHERE id = p_event_id), p_event_id, 'new_application',
    json_build_object('applicant_id', p_user_id, 'comment', p_comment));
END;
$$ LANGUAGE plpgsql;

-- Отмена заявки пользователем
CREATE OR REPLACE FUNCTION cancel_application(p_event_id UUID, p_user_id UUID)
RETURNS VOID AS $$
BEGIN
UPDATE registrations SET status = 'cancelled', updated_at = now()
WHERE event_id = p_event_id AND user_id = p_user_id;

PERFORM send_notification((SELECT organizer_id FROM events WHERE id = p_event_id), p_event_id, 'application_cancelled',
    json_build_object('applicant_id', p_user_id));
END;
$$ LANGUAGE plpgsql;

-- Подтверждение/отклонение заявки организатором
CREATE OR REPLACE FUNCTION organizer_confirm_registration(p_event_id UUID, p_user_id UUID, p_by_user UUID, p_confirm BOOLEAN)
RETURNS VOID AS $$
DECLARE
new_status registration_status_t;
BEGIN
  IF p_confirm THEN
    new_status := 'confirmed';
ELSE
    new_status := 'rejected';
END IF;

UPDATE registrations SET status = new_status, updated_at = now()
WHERE event_id = p_event_id AND user_id = p_user_id;

PERFORM send_notification(p_user_id, p_event_id, 'registration_' || new_status::text,
    json_build_object('event_id', p_event_id, 'by', p_by_user));
END;
$$ LANGUAGE plpgsql;

-- Модерация события админом
CREATE OR REPLACE FUNCTION admin_moderate_event(p_event_id UUID, p_admin_id UUID, p_action TEXT, p_reason TEXT)
RETURNS VOID AS $$
BEGIN
  IF p_action = 'approve' THEN
UPDATE events SET status = 'active', moderated_by = p_admin_id, moderated_at = now() WHERE id = p_event_id;
ELSIF p_action = 'reject' THEN
UPDATE events SET status = 'rejected', moderated_by = p_admin_id, moderated_at = now() WHERE id = p_event_id;
ELSE
    RAISE EXCEPTION 'Unknown moderation action';
END IF;

INSERT INTO moderation_logs(object_type, object_id, action, details, performed_by, performed_at)
VALUES ('event', p_event_id, p_action, json_build_object('reason', p_reason), p_admin_id, now());

PERFORM send_notification((SELECT organizer_id FROM events WHERE id = p_event_id), p_event_id, 'moderation_' || p_action,
    json_build_object('reason', p_reason));
END;
$$ LANGUAGE plpgsql;

-------------------------
-- 14. Триггеры
-------------------------
-- updated_at для users, events, registrations
DROP TRIGGER IF EXISTS trg_users_updated_at ON users;
CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION trigger_update_timestamp();

DROP TRIGGER IF EXISTS trg_events_updated_at ON events;
CREATE TRIGGER trg_events_updated_at
    BEFORE UPDATE ON events
    FOR EACH ROW EXECUTE FUNCTION trigger_update_timestamp();

DROP TRIGGER IF EXISTS trg_registrations_updated_at ON registrations;
CREATE TRIGGER trg_registrations_updated_at
    BEFORE UPDATE ON registrations
    FOR EACH ROW EXECUTE FUNCTION trigger_update_timestamp();

-- tsvector: обновление location_text_search в events (title + short_description + full_description)
CREATE OR REPLACE FUNCTION events_tsvector_update() RETURNS trigger AS $$
BEGIN
  NEW.location_text_search :=
    to_tsvector('russian', coalesce(NEW.title,'') || ' ' || coalesce(NEW.short_description,'') || ' ' || coalesce(NEW.full_description,''));
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_events_tsvector ON events;
CREATE TRIGGER trg_events_tsvector
    BEFORE INSERT OR UPDATE ON events
    FOR EACH ROW EXECUTE FUNCTION events_tsvector_update();


-- Поддержание participants_count в events (пересчитываем при изменениях регистраций)
CREATE OR REPLACE FUNCTION update_event_participants_count()
RETURNS TRIGGER AS $$
BEGIN
  -- Для INSERT/UPDATE/DELETE: берем event_id из NEW (или OLD для DELETE)
  IF TG_OP = 'DELETE' THEN
    PERFORM 1;
UPDATE events
SET participants_count = (
    SELECT COUNT(*) FROM registrations WHERE event_id = OLD.event_id AND status IN ('confirmed','attended')
)
WHERE id = OLD.event_id;
RETURN OLD;
ELSE
UPDATE events
SET participants_count = (
    SELECT COUNT(*) FROM registrations WHERE event_id = NEW.event_id AND status IN ('confirmed','attended')
)
WHERE id = NEW.event_id;
RETURN NEW;
END IF;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_registrations_participants_count ON registrations;
CREATE TRIGGER trg_registrations_participants_count
    AFTER INSERT OR UPDATE OR DELETE ON registrations
    FOR EACH ROW EXECUTE FUNCTION update_event_participants_count();


-- Пересчёт рейтинга после изменения reviews
CREATE OR REPLACE FUNCTION trg_reviews_recalc()
RETURNS TRIGGER AS $$
BEGIN
  IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
    PERFORM recalc_user_rating(NEW.author_id);
  ELSIF TG_OP = 'DELETE' THEN
    PERFORM recalc_user_rating(OLD.author_id);
END IF;
RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_reviews_after_change ON reviews;
CREATE TRIGGER trg_reviews_after_change
    AFTER INSERT OR UPDATE OR DELETE ON reviews
    FOR EACH ROW EXECUTE FUNCTION trg_reviews_recalc();


-------------------------
-- 15. Индексы и GIN / GIST / TRGM
-------------------------
-- users
-- CREATE INDEX idx_users_lastseen ON users(last_seen_at);

-- events: статусы, даты, категории
CREATE INDEX IF NOT EXISTS idx_events_status_starts ON events(status, starts_at);
CREATE INDEX IF NOT EXISTS idx_events_category ON events(category_id);
CREATE INDEX IF NOT EXISTS idx_events_importance ON events(importance);
-- tsvector индекс
CREATE INDEX IF NOT EXISTS idx_events_fts ON events USING GIN (location_text_search);
-- address trigram для быстрого LIKE
CREATE INDEX IF NOT EXISTS idx_events_address_trgm ON events USING GIN (address gin_trgm_ops);
-- fast lookup
CREATE INDEX IF NOT EXISTS idx_registrations_event ON registrations(event_id);
CREATE INDEX IF NOT EXISTS idx_registrations_user ON registrations(user_id);
CREATE INDEX IF NOT EXISTS idx_reviews_event ON reviews(event_id);

-------------------------
-- 16. Примеры GRANT (рекомендуется настраивать роли в production)
-------------------------
-- Создадим две роли приложения-уровня (пример)
-- DO NOT run these if у вас другая схема ролей; корректируйте под окружение.
DO $$
BEGIN
  -- create roles if not exist
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_reader') THEN
CREATE ROLE app_reader NOINHERIT;
END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_writer') THEN
CREATE ROLE app_writer NOINHERIT;
END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_migrator') THEN
CREATE ROLE app_migrator NOINHERIT;
END IF;
END;
$$ LANGUAGE plpgsql;

-- Примерные GRANT'ы (нативные права)
GRANT SELECT ON TABLE users, events, categories, tags, event_tags, registrations, reviews, notifications, user_ratings TO app_reader;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE users, events, categories, tags, event_tags, registrations, reviews, notifications, user_ratings TO app_writer;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO app_migrator;

COMMIT;

