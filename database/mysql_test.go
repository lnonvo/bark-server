package database

import "testing"

func TestMySQLDeviceKeyQueries(t *testing.T) {
	tests := []struct {
		column string
		want   []string
	}{
		{
			column: "device_key",
			want: []string{
				"SELECT `token` FROM `devices` WHERE `device_key`=? ORDER BY `id` DESC LIMIT 1",
				"SELECT COUNT(1) FROM `devices` WHERE `device_key`=?",
				"UPDATE `devices` SET `token`=? WHERE `device_key`=?",
				"INSERT INTO `devices` (`device_key`,`token`) VALUES (?,?) ON DUPLICATE KEY UPDATE `token`=?",
				"DELETE FROM `devices` WHERE `device_key`=?",
			},
		},
		{
			column: "key",
			want: []string{
				"SELECT `token` FROM `devices` WHERE `key`=? ORDER BY `id` DESC LIMIT 1",
				"SELECT COUNT(1) FROM `devices` WHERE `key`=?",
				"UPDATE `devices` SET `token`=? WHERE `key`=?",
				"INSERT INTO `devices` (`key`,`token`) VALUES (?,?) ON DUPLICATE KEY UPDATE `token`=?",
				"DELETE FROM `devices` WHERE `key`=?",
			},
		},
	}

	for _, test := range tests {
		mysqlDeviceKeyColumn = test.column
		queries := []string{
			mysqlDeviceTokenByKeyQuery(),
			mysqlDeviceExistsByKeyQuery(),
			mysqlUpdateDeviceTokenByKeyQuery(),
			mysqlInsertDeviceTokenByKeyQuery(),
			mysqlDeleteDeviceByKeyQuery(),
		}
		for i, query := range queries {
			if query != test.want[i] {
				t.Errorf("column %q query %d = %q, want %q", test.column, i, query, test.want[i])
			}
		}
	}

	mysqlDeviceKeyColumn = "device_key"
}

func TestMySQLMessageInsertQuery(t *testing.T) {
	want := "INSERT INTO `message` (`created_by`,`created_time`,`updated_by`,`updated_time`,`version`,`deleted`,`content`) VALUES (?,?,?,?,?,?,?)"
	if got := mysqlMessageInsertQuery(); got != want {
		t.Fatalf("mysqlMessageInsertQuery() = %q, want %q", got, want)
	}
}

func TestBuildMessageContent(t *testing.T) {
	tests := []struct {
		name                  string
		title, subtitle, body string
		want                  string
	}{
		{name: "body", title: "title", subtitle: "subtitle", body: "body", want: "body"},
		{name: "title and subtitle", title: "title", subtitle: "subtitle", want: "title\nsubtitle"},
		{name: "title", title: "title", want: "title"},
		{name: "subtitle", subtitle: "subtitle", want: "subtitle"},
		{name: "empty", want: "Empty Message"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := BuildMessageContent(test.title, test.subtitle, test.body); got != test.want {
				t.Fatalf("BuildMessageContent() = %q, want %q", got, test.want)
			}
		})
	}
}
