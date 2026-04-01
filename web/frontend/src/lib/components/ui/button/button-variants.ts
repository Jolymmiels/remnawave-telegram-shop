import { tv } from "tailwind-variants"

export const buttonVariants = tv({
  base: "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-all outline-none disabled:pointer-events-none disabled:opacity-50 focus-visible:ring-2 focus-visible:ring-(--ring) focus-visible:ring-offset-2 focus-visible:ring-offset-(--background) [&_svg]:pointer-events-none [&_svg]:size-4 shrink-0",
  variants: {
    variant: {
      default: "bg-(--primary) text-(--primary-foreground) hover:opacity-90",
      secondary: "bg-(--secondary) text-(--secondary-foreground) hover:bg-(--accent)",
      outline:
        "border border-(--input) bg-(--background) hover:bg-(--accent) hover:text-(--accent-foreground)",
      ghost: "hover:bg-(--accent) hover:text-(--accent-foreground)",
      destructive: "bg-(--destructive) text-white hover:opacity-90"
    },
    size: {
      default: "h-10 px-4 py-2",
      sm: "h-9 rounded-md px-3",
      lg: "h-11 rounded-md px-8",
      icon: "size-10"
    }
  },
  defaultVariants: {
    variant: "default",
    size: "default"
  }
})
