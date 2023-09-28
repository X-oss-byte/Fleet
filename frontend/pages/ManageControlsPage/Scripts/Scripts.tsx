import React, { useContext, useRef, useState } from "react";

import { createMockScript } from "__mocks__/scriptMock";
import { AppContext } from "context/app";
import { IScript } from "services/entities/scripts";

import CustomLink from "components/CustomLink";

import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import ScriptListHeading from "./components/ScriptListHeading";
import ScriptListItem from "./components/ScriptListItem";
import DeleteScriptModal from "./components/DeleteScriptModal";
import RerunScriptModal from "./components/RerunScriptModal";
import FileUploader from "../components/FileUploader";
import UploadList from "../components/UploadList";

// TODO: remove when get integrate with API.
const scripts: IScript[] = [createMockScript()];

const baseClass = "scripts";

const Scripts = () => {
  const { isPremiumTier } = useContext(AppContext);
  const [showRerunScriptModal, setShowRerunScriptModal] = useState(false);
  const [showDeleteScriptModal, setShowDeleteScriptModal] = useState(false);

  const selectedScript = useRef<IScript | null>(null);

  // The user is not a premium tier, so show the premium feature message.
  if (!isPremiumTier) {
    return (
      <PremiumFeatureMessage
        className={`${baseClass}__premium-feature-message`}
      />
    );
  }

  const onClickDelete = (script: IScript) => {
    selectedScript.current = script;
    setShowDeleteScriptModal(true);
  };

  const onCancelRerun = () => {
    selectedScript.current = null;
    setShowRerunScriptModal(false);
  };

  const onCancelDelete = () => {
    selectedScript.current = null;
    setShowDeleteScriptModal(false);
  };

  // TODO: change when integrating with API
  const onRerunScript = (scriptId: number) => {
    console.log("rerun", scriptId);
    setShowRerunScriptModal(false);
  };

  // TODO: change when integrating with API
  const onDeleteScript = (scriptId: number) => {
    console.log("delete", scriptId);
    setShowDeleteScriptModal(false);
  };

  return (
    <div className={baseClass}>
      <p className={`${baseClass}__description`}>
        Upload scripts to change configuration and remediate issues on macOS
        hosts. You can run scripts on individual hosts.{" "}
        <CustomLink
          text="Learn more"
          url="https://fleetdm.com/docs/using-fleet/scripts"
          newTab
        />
      </p>
      {scripts.length !== 0 && (
        <UploadList
          listItems={scripts}
          HeadingComponent={ScriptListHeading}
          ListItemComponent={({ listItem }) => (
            <ScriptListItem script={listItem} onDelete={onClickDelete} />
          )}
        />
      )}
      <FileUploader
        className={`${baseClass}__script-file-uploader`}
        icon="file-bash"
        message="Script (.sh)"
        additionalInfo="Script will run with “#!/bin/sh”."
        accept=".sh"
        onFileUpload={() => {
          return null;
        }}
      />
      {showRerunScriptModal && selectedScript.current && (
        <RerunScriptModal
          scriptName={selectedScript.current?.name}
          scriptId={selectedScript.current?.id}
          onCancel={onCancelRerun}
          onRerun={onRerunScript}
        />
      )}
      {showDeleteScriptModal && selectedScript.current && (
        <DeleteScriptModal
          scriptName={selectedScript.current?.name}
          scriptId={selectedScript.current?.id}
          onCancel={onCancelDelete}
          onDelete={onDeleteScript}
        />
      )}
    </div>
  );
};

export default Scripts;
