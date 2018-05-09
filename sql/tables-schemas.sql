CREATE TABLE task_hours_bid_and_members (
  id int(11) NOT NULL AUTO_INCREMENT,
  task_id int(11) NOT NULL,
  member_identity varchar(128) NOT NULL,
  member_time_bid decimal(10, 0) NOT NULL,
  member_nick varchar(128) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE INDEX task_hours_bid_and_members_id_uindex (id)
)
ENGINE = INNODB
CHARACTER SET utf8mb4
COLLATE utf8mb4_general_ci
COMMENT = 'Содержит оценки участников голосования и своего рода идентификаторы участников голосования';

CREATE TABLE tasks (
  task_id int(11) NOT NULL AUTO_INCREMENT,
  task_title text NOT NULL,
  task_bidding_done tinyint(1) DEFAULT 0,
  PRIMARY KEY (task_id),
  UNIQUE INDEX tasks_task_id_uindex (task_id)
)
ENGINE = INNODB
CHARACTER SET utf8mb4
COLLATE utf8mb4_general_ci;

CREATE TABLE slack_tokens (
  slack_token varchar(128) NOT NULL,
  PRIMARY KEY (slack_token)
)
  ENGINE = INNODB
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_general_ci;

CREATE TABLE last_task (
  id int(11) NOT NULL COMMENT 'Идентификатор записи в этой таблице. Всегда должен быть только один.',
  task_id int(11) DEFAULT NULL COMMENT 'Идентификатор последней задачи равный task_id в таблице tasks'
)
ENGINE = INNODB
CHARACTER SET utf8mb4
COLLATE utf8mb4_general_ci
COMMENT = 'Содержит идентификатор последней задачи для которой проводилось голосование';