-- ==========================
-- USER PROFILES
-- ==========================
CREATE TABLE public.user_profiles (
    id uuid PRIMARY KEY,
    username text NOT NULL UNIQUE,
    password text,
    role text NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
    email text UNIQUE,
    full_name text,
    avatar_url text,
    provider text DEFAULT 'local',
    provider_id text,
    status text DEFAULT 'active',
    email_verified_at timestamptz,
    remember_token text,
    last_login timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz DEFAULT now()
);

-- ==========================
-- WORKSPACES
-- ==========================
CREATE TABLE public.workspaces (
    id uuid PRIMARY KEY,
    name text NOT NULL,
    slug text NOT NULL UNIQUE,
    owner_id uuid NOT NULL,
    description text,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),

    CONSTRAINT fk_workspace_owner
        FOREIGN KEY (owner_id)
        REFERENCES user_profiles(id)
        ON DELETE CASCADE
);

-- ==========================
-- WORKSPACE MEMBERS
-- ==========================
CREATE TABLE public.workspace_members (
    workspace_id uuid NOT NULL,
    user_id uuid NOT NULL,
    role text NOT NULL,
    joined_at timestamptz DEFAULT now(),

    PRIMARY KEY (workspace_id, user_id),

    CONSTRAINT fk_member_workspace
        FOREIGN KEY (workspace_id)
        REFERENCES workspaces(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_member_user
        FOREIGN KEY (user_id)
        REFERENCES user_profiles(id)
        ON DELETE CASCADE,

    CONSTRAINT check_member_role
        CHECK (role IN ('owner', 'editor', 'viewer'))
);

-- ==========================
-- PROJECTS
-- ==========================
CREATE TABLE public.projects (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL,
    name text NOT NULL,
    description text,
    created_by uuid,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),

    CONSTRAINT fk_project_workspace
        FOREIGN KEY (workspace_id)
        REFERENCES workspaces(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_project_creator
        FOREIGN KEY (created_by)
        REFERENCES user_profiles(id)
        ON DELETE SET NULL
);

-- ==========================
-- DOCUMENTS
-- ==========================
CREATE TABLE public.documents (
    id uuid PRIMARY KEY,
    project_id uuid,
    workspace_id uuid NOT NULL,
    title text NOT NULL,
    diagram_type text NOT NULL CHECK (diagram_type IN ('flowchart', 'erd', 'usecase', 'sequence', 'mindmap', 'blank')),
    content jsonb NOT NULL,
    view jsonb NOT NULL,
    version integer DEFAULT 1,
    created_by uuid,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),

    CONSTRAINT fk_document_project
        FOREIGN KEY (project_id)
        REFERENCES projects(id)
        ON DELETE SET NULL,

    CONSTRAINT fk_document_workspace
        FOREIGN KEY (workspace_id)
        REFERENCES workspaces(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_document_creator
        FOREIGN KEY (created_by)
        REFERENCES user_profiles(id)
        ON DELETE SET NULL
);

-- ==========================
-- DOCUMENT VERSIONS
-- ==========================
CREATE TABLE public.document_versions (
    id uuid PRIMARY KEY,
    document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    version integer NOT NULL,
    content jsonb NOT NULL,
    view jsonb NOT NULL,
    created_by uuid REFERENCES user_profiles(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_document_versions_document_id ON document_versions(document_id);
