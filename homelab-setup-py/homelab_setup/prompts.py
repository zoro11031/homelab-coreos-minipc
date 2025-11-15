"""
Interactive prompt utilities using questionary.

Provides consistent user interaction prompts throughout the setup process.
"""

from typing import Optional, List
import questionary
from questionary import Style

# Custom style for prompts
custom_style = Style(
    [
        ("qmark", "fg:#00d7ff bold"),  # Question mark
        ("question", "bold"),  # Question text
        ("answer", "fg:#00d7ff bold"),  # Answer text
        ("pointer", "fg:#00d7ff bold"),  # Pointer for selection
        ("highlighted", "fg:#00d7ff bold"),  # Highlighted option
        ("selected", "fg:#00d7ff"),  # Selected option
        ("separator", "fg:#6c6c6c"),  # Separator
        ("instruction", ""),  # Instruction
        ("text", ""),  # Normal text
        ("disabled", "fg:#858585 italic"),  # Disabled option
    ]
)


def confirm(message: str, default: bool = True) -> bool:
    """
    Ask a yes/no question.

    Args:
        message: Question to ask
        default: Default answer

    Returns:
        True for yes, False for no
    """
    return questionary.confirm(message, default=default, style=custom_style).ask()


def prompt_text(message: str, default: str = "") -> str:
    """
    Prompt for text input.

    Args:
        message: Prompt message
        default: Default value

    Returns:
        User input string
    """
    return questionary.text(message, default=default, style=custom_style).ask()


def prompt_password(message: str = "Password") -> str:
    """
    Prompt for password input (hidden).

    Args:
        message: Prompt message

    Returns:
        Password string
    """
    while True:
        password1 = questionary.password(message, style=custom_style).ask()
        password2 = questionary.password("Confirm password", style=custom_style).ask()

        if password1 == password2:
            if password1:
                return password1
            else:
                print("Password cannot be empty.")
        else:
            print("Passwords do not match. Please try again.")


def prompt_select(message: str, choices: List[str], default: Optional[str] = None) -> str:
    """
    Prompt user to select from a list of choices.

    Args:
        message: Prompt message
        choices: List of choices
        default: Default selection

    Returns:
        Selected choice
    """
    return questionary.select(
        message, choices=choices, default=default, style=custom_style
    ).ask()


def prompt_checkbox(message: str, choices: List[str]) -> List[str]:
    """
    Prompt user to select multiple items from a list.

    Args:
        message: Prompt message
        choices: List of choices

    Returns:
        List of selected choices
    """
    return questionary.checkbox(message, choices=choices, style=custom_style).ask()


def prompt_path(message: str, default: str = "") -> str:
    """
    Prompt for a file path with autocomplete.

    Args:
        message: Prompt message
        default: Default path

    Returns:
        Path string
    """
    return questionary.path(message, default=default, style=custom_style).ask()
