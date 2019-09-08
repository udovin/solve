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

export interface Problem {
	ID: number;
	UserID: number;
	CreateTime: number;
	Title: string;
	Description: string;
}

export interface ContestProblem extends Problem {
	Code: string;
}

export interface Contest {
	ID: number;
	UserID: number;
	CreateTime: number;
	Title: string;
	Problems: ContestProblem[];
}

export interface Compiler {
	ID: number;
	Name: string;
	CreateTime: number;
}

export interface Solution {
	ID: number;
	ProblemID: number;
	ContestID?: number;
	CompilerID: number;
	UserID: number;
	SourceCode: string;
	CreateTime: number;
}