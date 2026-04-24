import { login, register } from "../lib/auth";

function initAuthPage() {
  const tabs = Array.from(document.querySelectorAll(".auth-tab"));
  const forms = Array.from(document.querySelectorAll(".auth-form"));
  const message = document.getElementById("auth-message");
  const loginForm = document.getElementById("login-form");
  const registerForm = document.getElementById("register-form");
  const switchToLoginButton = document.getElementById("switch-to-login");
  const loginSubmit = document.getElementById("login-submit");
  const registerSubmit = document.getElementById("register-submit");

  function getInputValue(id: string): string {
    const input = document.getElementById(id);
    return input instanceof HTMLInputElement ? input.value.trim() : "";
  }

  function setBusy(
    button: HTMLElement | null,
    busy: boolean,
    busyLabel: string,
    idleLabel: string
  ): void {
    if (!(button instanceof HTMLButtonElement)) return;
    button.disabled = busy;
    button.textContent = busy ? busyLabel : idleLabel;
  }

  function showMessage(text: string, tone: "success" | "error"): void {
    if (!message) return;
    message.textContent = text;
    message.className = `auth-message is-visible ${tone === "success" ? "is-success" : "is-error"}`;
  }

  function clearMessage(): void {
    if (!message) return;
    message.className = "auth-message";
    message.textContent = "";
  }

  function activate(targetId: string | undefined, updateHash = true): void {
    const normalizedTarget = targetId === "register-form" ? "register-form" : "login-form";

    tabs.forEach((tab) => {
      const isActive = tab instanceof HTMLElement && tab.dataset.target === normalizedTarget;
      tab.classList.toggle("is-active", isActive);
      if (tab instanceof HTMLElement) {
        tab.setAttribute("aria-selected", isActive ? "true" : "false");
      }
    });

    forms.forEach((form) => {
      form.classList.toggle("is-active", form.id === normalizedTarget);
    });

    if (updateHash) {
      window.location.hash = normalizedTarget;
    }

    clearMessage();
  }

  function syncFromHash(): void {
    activate(window.location.hash === "#register-form" ? "register-form" : "login-form", false);
  }

  function showError(prefix: string, error: unknown): void {
    if (error instanceof Error && error.message) {
      showMessage(`${prefix}: ${error.message}`, "error");
      return;
    }

    showMessage(`${prefix}: ocorreu um erro inesperado.`, "error");
  }

  tabs.forEach((tab) => {
    tab.addEventListener("click", (event) => {
      event.preventDefault();
      if (tab instanceof HTMLElement) {
        activate(tab.dataset.target);
      }
    });
  });

  window.addEventListener("hashchange", syncFromHash);
  switchToLoginButton?.addEventListener("click", () => activate("login-form"));
  syncFromHash();

  loginForm?.addEventListener("submit", async (event) => {
    event.preventDefault();

    const email = getInputValue("login-email");
    const passwordInput = document.getElementById("login-password");
    const password = passwordInput instanceof HTMLInputElement ? passwordInput.value : "";

    if (!email || !password) {
      showMessage("Preencha email e senha para entrar.", "error");
      return;
    }

    try {
      setBusy(loginSubmit, true, "Entrando...", "Entrar");
      showMessage("Autenticando...", "success");
      await login(email, password);
      showMessage("Login concluido. Redirecionando para organizacoes...", "success");
      window.location.href = "/organizations";
    } catch (error) {
      showError("Erro ao entrar", error);
    } finally {
      setBusy(loginSubmit, false, "Entrando...", "Entrar");
    }
  });

  registerForm?.addEventListener("submit", async (event) => {
    event.preventDefault();

    const name = getInputValue("register-name");
    const email = getInputValue("register-email");
    const passwordInput = document.getElementById("register-password");
    const confirmInput = document.getElementById("register-password-confirm");
    const password = passwordInput instanceof HTMLInputElement ? passwordInput.value : "";
    const passwordConfirm = confirmInput instanceof HTMLInputElement ? confirmInput.value : "";

    if (!name || !email || !password || !passwordConfirm) {
      showMessage("Preencha todos os campos do cadastro.", "error");
      return;
    }

    if (password.length < 6) {
      showMessage("A senha deve ter pelo menos 6 caracteres.", "error");
      return;
    }

    if (password !== passwordConfirm) {
      showMessage("A confirmacao de senha nao confere.", "error");
      return;
    }

    try {
      setBusy(registerSubmit, true, "Criando conta...", "Criar conta");
      showMessage("Criando conta...", "success");
      await register(name, email, password);

      const loginEmailInput = document.getElementById("login-email");
      if (loginEmailInput instanceof HTMLInputElement) {
        loginEmailInput.value = email;
      }

      if (passwordInput instanceof HTMLInputElement) {
        passwordInput.value = "";
      }

      if (confirmInput instanceof HTMLInputElement) {
        confirmInput.value = "";
      }

      activate("login-form");
      showMessage("Conta criada com sucesso. Agora faca login.", "success");
    } catch (error) {
      showError("Erro ao cadastrar", error);
    } finally {
      setBusy(registerSubmit, false, "Criando conta...", "Criar conta");
    }
  });
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initAuthPage, { once: true });
} else {
  initAuthPage();
}
