"use client";

import { usePathname } from "next/navigation";
import { Navbar } from "./Navbar";

export const NavbarWrapper = () => {
  const pathname = usePathname();
  const isWatchPage = pathname?.includes("/videos/watch/");

  if (isWatchPage) return null;

  return <Navbar />;
};
