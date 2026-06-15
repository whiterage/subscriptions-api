create table if not exists subscriptions (
	id bigserial primary key,
	service_name text not null,
	price integer not null check (price > 0),
	user_id uuid not null,
	start_date date not null,
	end_date date,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	check (end_date is null or end_date >= start_date)
);

create index if not exists idx_subscriptions_user_id on subscriptions (user_id);
create index if not exists idx_subscriptions_service_name on subscriptions (service_name);
create index if not exists idx_subscriptions_period on subscriptions (start_date, end_date);
