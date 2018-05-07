create table task_hours_bid_and_memebers
(
  task_id         int     not null,
  member_idenity  text    not null,
  member_time_bid decimal not null
)
  comment 'Содержит оценки участников голосования и своего рода идентификаторы участников голосования';

create table tasks
(
  task_id           int auto_increment
    primary key,
  task_title        text             not null,
  task_bidding_done bit default b'0' null,
  constraint tasks_task_id_uindex
  unique (task_id)
);

