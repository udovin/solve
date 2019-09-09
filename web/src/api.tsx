export type User = {
	ID: number;
	Login: string;
	CreateTime: number;
};

export type Session = {
	ID: number;
	UserID: number;
	CreateTime: number;
	ExpireTime: number;
};

export type CurrentSession = Session & {
	User: User;
};

export type Problem = {
	ID: number;
	UserID: number;
	CreateTime: number;
	Title: string;
	Description: string;
	Solutions?: Solution[];
};

export type ContestProblem = Problem & {
	Code: string;
};

export type Contest = {
	ID: number;
	UserID: number;
	CreateTime: number;
	Title: string;
	Problems: ContestProblem[];
};

export type Compiler = {
	ID: number;
	Name: string;
	CreateTime: number;
};

export type Solution = {
	ID: number;
	ProblemID: number;
	ContestID?: number;
	CompilerID: number;
	UserID: number;
	SourceCode: string;
	CreateTime: number;
};

export type Report = {
	ID: number;
	SolutionID: number;
	Verdict: number;
	CreateTime: number;
};
