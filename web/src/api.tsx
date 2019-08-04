export interface User {
	ID: number;
	Login: string;
	CreateTime: number;
}

export interface Session {
	ID: number;
	UserID: number;
	CreateTime: number;
	ExpireTime: number;
}

export interface CurrentSession extends Session {
	User: User;
}

export interface Contest {
	ID: number;
	OwnerID: number;
	CreateTime: number;
}
