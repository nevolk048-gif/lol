const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

class ApiClient {
  private accessToken: string | null = null;

  setToken(token: string | null) {
    this.accessToken = token;
    if (typeof window !== "undefined") {
      if (token) localStorage.setItem("access_token", token);
      else localStorage.removeItem("access_token");
    }
  }

  getToken(): string | null {
    if (this.accessToken) return this.accessToken;
    if (typeof window !== "undefined") {
      // Check both access_token and auth-storage (Zustand)
      const token = localStorage.getItem("access_token");
      if (token) return token;

      const authStorage = localStorage.getItem("auth-storage");
      if (authStorage) {
        try {
          const parsed = JSON.parse(authStorage);
          return parsed.state?.accessToken || null;
        } catch {
          return null;
        }
      }
    }
    return null;
  }

  private async request<T>(
    path: string,
    options: RequestInit = {}
  ): Promise<T> {
    const token = this.getToken();
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      ...(options.headers as Record<string, string>),
    };
    if (token) headers.Authorization = `Bearer ${token}`;

    const res = await fetch(`${API_URL}/api/v1${path}`, {
      ...options,
      headers,
    });

    const json = await res.json();
    if (!res.ok || !json.success) {
      throw new Error(json.error?.message || "Request failed");
    }
    return json.data ?? json;
  }

  // Auth
  login(email: string, password: string) {
    return this.request<import("@/types").LoginResponse>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    });
  }

  me() {
    return this.request<import("@/types").User>("/auth/me");
  }

  logout(refreshToken: string) {
    return this.request("/auth/logout", {
      method: "POST",
      body: JSON.stringify({ refresh_token: refreshToken }),
    });
  }

  // Dashboard
  getDashboard(isSandbox?: boolean) {
    const q = isSandbox !== undefined ? `?is_sandbox=${isSandbox}` : "";
    return this.request<import("@/types").DashboardData>(`/dashboard${q}`);
  }

  getFinance() {
    return this.request<import("@/types").FinanceData>("/finance");
  }

  getMonitoring() {
    return this.request<import("@/types").MonitoringData>("/monitoring");
  }

  // Transactions
  getTransactions(params?: Record<string, string>) {
    const q = params ? "?" + new URLSearchParams(params).toString() : "";
    return this.request<import("@/types").Transaction[]>(`/transactions${q}`);
  }

  getTransaction(id: string) {
    return this.request<import("@/types").Transaction>(`/transactions/${id}`);
  }

  // Providers
  getProviders() {
    return this.request<import("@/types").Provider[]>("/providers");
  }

  getProvider(id: string) {
    return this.request<import("@/types").Provider>(`/providers/${id}`);
  }

  createProvider(data: { name: string; webhook_url?: string; is_sandbox?: boolean }) {
    return this.request<import("@/types").Provider>("/providers", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  deleteProvider(id: string) {
    return this.request(`/providers/${id}`, { method: "DELETE" });
  }

  // Casinos
  getCasinos() {
    return this.request<import("@/types").Casino[]>("/casinos");
  }

  getCasino(id: string) {
    return this.request<import("@/types").Casino>(`/casinos/${id}`);
  }

  createCasino(data: { name: string; webhook_url?: string; is_sandbox?: boolean }) {
    return this.request<import("@/types").Casino>("/casinos", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  deleteCasino(id: string) {
    return this.request(`/casinos/${id}`, { method: "DELETE" });
  }

  // Requisites
  getRequisites(providerId?: string) {
    const q = providerId ? `?provider_id=${providerId}` : "";
    return this.request<import("@/types").Requisite[]>(`/requisites${q}`);
  }

  createRequisite(data: Partial<import("@/types").Requisite> & { provider_id: string; daily_limit: number }) {
    return this.request<import("@/types").Requisite>("/requisites", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  // Routing
  getRouteRules() {
    return this.request<import("@/types").RouteRule[]>("/routing/rules");
  }

  createRouteRule(data: Partial<import("@/types").RouteRule> & { provider_id: string; weight: number }) {
    return this.request<import("@/types").RouteRule>("/routing/rules", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  deleteRouteRule(id: string) {
    return this.request(`/routing/rules/${id}`, { method: "DELETE" });
  }

  // Users
  getUsers() {
    return this.request<import("@/types").User[]>("/users");
  }

  createUser(data: { email: string; password: string; role: string }) {
    return this.request<import("@/types").User>("/users", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  // Audit & Integration logs
  getAuditLogs(params?: Record<string, string>) {
    const q = params ? "?" + new URLSearchParams(params).toString() : "";
    return this.request<import("@/types").AuditLog[]>(`/audit-logs${q}`);
  }

  getIntegrationLogs(params?: Record<string, string>) {
    const q = params ? "?" + new URLSearchParams(params).toString() : "";
    return this.request<import("@/types").IntegrationLog[]>(`/integration-logs${q}`);
  }

  // Sandbox
  sandboxSetup() {
    return this.request("/sandbox/setup", { method: "POST" });
  }

  sandboxDeposit(casinoId: string, amount: number) {
    return this.request("/sandbox/deposit", {
      method: "POST",
      body: JSON.stringify({ casino_id: casinoId, amount }),
    });
  }

  sandboxGenerateStats() {
    return this.request<{ message: string }>("/sandbox/generate-stats", { method: "POST" });
  }

  sandboxGenerateTraffic(casinoId: string, count: number) {
    return this.request("/sandbox/generate-traffic", {
      method: "POST",
      body: JSON.stringify({ casino_id: casinoId, count }),
    });
  }
}

export const api = new ApiClient();
