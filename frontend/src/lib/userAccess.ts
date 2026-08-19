export type UserRole = "owner" | "admin" | "support" | "user";

export interface User {
  user_id: number;
  email: string;
  is_admin: boolean;
  role: UserRole;
  created_at: string;
}

export const getUserRole = (user: User): UserRole =>
  user.role || (user.is_admin ? "owner" : "user");

export const hasStaffAccess = (user: User | null) =>
  Boolean(user && getUserRole(user) !== "user");
