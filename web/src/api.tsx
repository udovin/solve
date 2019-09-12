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

export type Report = {
	ID: number;
	SolutionID: number;
	Verdict: number;
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
	Report?: Report;
};

export const RUNNING: number = -1;
export const QUEUED: number = 0;
export const ACCEPTED: number = 1;
export const COMPILATION_ERROR: number = 2;
export const TIME_LIMIT_EXCEEDED: number = 3;
export const MEMORY_LIMIT_EXCEEDED: number = 4;
export const RUNTIME_ERROR: number = 5;
export const WRONG_ANSWER: number = 6;
export const PRESENTATION_ERROR: number = 7;

export const getShortVerdict = (verdict: number) => {
	switch (verdict) {
		case RUNNING:
		case QUEUED:
			return "?";
		case ACCEPTED:
			return "AC";
		case COMPILATION_ERROR:
			return "CE";
		case TIME_LIMIT_EXCEEDED:
			return "TL";
		case MEMORY_LIMIT_EXCEEDED:
			return "ML";
		case RUNTIME_ERROR:
			return "RE";
		case WRONG_ANSWER:
			return "WA";
		case PRESENTATION_ERROR:
			return "PE";
	}
	return "?";
};
