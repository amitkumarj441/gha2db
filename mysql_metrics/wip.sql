select
  count(distinct a.id) as result
from
  gha_events e,
  gha_actors a
where
  e.id in (
    select
      min(ev.id)
    from
      gha_issues_labels il,
      gha_events ev
    where
      ev.id = il.event_id
      and ev.created_at >= '2017-08-16 00:00:00' and ev.created_at < '2017-08-17 00:00:00'
      and il.label_id in (
        select id from gha_labels where name in ('lgtm', 'LGTM')
      )
    group by issue_id
    union
    select
      ev.id
    from
      gha_texts t,
      gha_events ev
    where
      ev.id = t.event_id
      and ev.created_at >= '2017-08-16 00:00:00' and ev.created_at < '2017-08-17 00:00:00'
      and preg_rlike('{^\\s*lgtm\\s*$}i', t.body)
  )
and e.actor_id = a.id
and a.login not in ('googlebot')
and a.login not like 'k8s-%'

    select
      ev.id
    from
      gha_issues_labels il,
      gha_events ev
    where
      ev.id = il.event_id
      and ev.created_at >= '2017-08-16 00:00:00' and ev.created_at < '2017-08-17 00:00:00'